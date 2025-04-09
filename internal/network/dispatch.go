package network

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"go-chat/pkg/crypt"
	"time"
)

var handlers = map[SignalType]func(*Network, Signal){
	SignalTypeNeedNewbieInvite: processNeedNewbieInvite,
	SignalTypeSendInvite:       processSignalTypeSendInvite,
	SignalTypeInvite:           processSignalTypeInvite,
	SignalTypeOffer:            processSignalTypeOffer,
	SignalTypeAnswer:           processSignalTypeAnswer,
	SignalTypeConnectionSecret: processSignalTypeConnectionSecret,
	SignalTypeConnectionProof:  processSignalTypeConnectionProof,
	SignalTypeTrusted:          processSignalTypeTrusted,
}

func (n *Network) dispatch(in <-chan Signal) {
	for s := range in {
		h, ok := handlers[s.Type]
		if !ok {
			return
		}
		h(n, s)
	}
}

func processNeedNewbieInvite(n *Network, in Signal) {
	if n.freeSlots.Add(1) > maxFreeSlotsCount {
		n.freeSlots.Add(-1)
		n.broadcast(in)
		return
	}

	var inh Handshake
	err := inh.Unmarshal(in.Payload)
	if err != nil {
		return
	}

	n.queueeMu.Lock()
	defer n.queueeMu.Unlock()

	n.queuee[in.Author] = &peer{
		pubKey:    inh.PubKey,
		signature: inh.PubSign,
	}

	outh := Handshake{
		PubKey:  n.privKey.PublicKey(),
		PubSign: n.pubSignature,
	}

	sum := sha256.Sum256(inh.PubKey.Bytes())
	recipient := hex.EncodeToString(sum[:])
	toNewbie := newSignal(SignalTypeReadyToInvite, recipient, n.hash, outh.Marshal())

	readyToInvite := ReadyToInviteNewbie{
		ConnectionSecret: rand.Text(),
		Signal:           toNewbie,
	}

	out := newSignal(SignalTypeRedyToInviteNewbie, in.Author, n.hash, readyToInvite.Marshal())

	n.broadcast(out)
}

func processReadyToInviteNewbie(n *Network, in Signal) {
	if in.Recipient != n.hash {
		n.broadcast(in)
		return
	}

	var rinv ReadyToInviteNewbie
	err := rinv.Unmarshal(in.Payload)
	if err != nil {
		return
	}

	n.queueeMu.RLock()
	defer n.queueeMu.RUnlock()
	inviteWaiter, ok := n.onboarding[rinv.Signal.Recipient]
	if !ok {
		return
	}

	inviteWaiter.mu.Lock()
	defer inviteWaiter.mu.Unlock()
	if len(inviteWaiter.secrets) > reqConns {
		return
	}

	inviteWaiter.secrets[rinv.Author] = rinv.ConnectionSecret

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = inviteWaiter.peer.Send(ctx, rinv.Signal)
	if err != nil {
		return
	}
}

func processReadyToInvite(n *Network, in Signal) {
	if in.Recipient != n.hash {
		n.broadcast(in)
		return
	}

	var h Handshake
	err := h.Unmarshal(in.Payload)
	if err != nil {
		return
	}

	n.queueeMu.Lock()
	defer n.queueeMu.Unlock()
	n.queuee[in.Author] = &peer{
		pubKey:    h.PubKey,
		signature: h.PubSign,
	}

	encryptedSecret, err := crypt.Encrypt([]byte(rand.Text()), n.privKey, h.PubKey)
	if err != nil {
		return
	}

	signature := ed25519.Sign(n.privSign, encryptedSecret)
	payload := append(encryptedSecret, signature...)
	out := newSignal(SignalTypeWaitOffer, in.Author, n.hash, payload)

	n.broadcast(out)
}

func processWaitOffer(n *Network, in Signal) {
	if n.hash != in.Recipient {
		n.broadcast(in)
		return
	}

	n.queueeMu.Lock()
	defer n.queueeMu.Unlock()
	offerWaiter, ok := n.queuee[in.Author]
	if !ok {
		return
	}

	sigStart := len(in.Payload) - 64
	secret := in.Payload[:sigStart]
	signature := in.Payload[sigStart:]
	if !ed25519.Verify(offerWaiter.signature, secret, signature) {
		return
	}

	decryptedSecret, err := crypt.Decrypt(secret, n.privKey, offerWaiter.pubKey)
	if err != nil {
		return
	}

}

func processSignalTypeNeedInvite2(n *Network, s Signal) {
	if n.freeSlots.Add(1) > maxFreeSlotsCount {
		n.freeSlots.Add(-1)
		n.broadcast(s)
		return
	}

	secret := make([]byte, 12)
	for written := 0; written < 12; {
		n, err := rand.Read(secret)
		if err != nil {
			return
		}
		written += n
	}

	sign := make([]byte, 32)
	for written := 0; written < 12; {
		n, err := rand.Read(sign)
		if err != nil {
			return
		}
		written += n
	}

	pubKey, err := ecdh.P256().NewPublicKey(s.Payload[:65])
	if err != nil {
		return
	}

	signature := ed25519.PublicKey(s.Payload[65:])
	n.queueeMu.Lock()
	defer n.queueeMu.Unlock()
	sum := sha256.Sum256(s.Payload[:65])
	n.queuee[hex.EncodeToString(sum[:])] = newPeer(pubKey, signature)
	_, err = crypt.Encrypt(sign, n.privKey, pubKey)
	if err != nil {
		return
	}
}

func processSignalTypeSendInvite(n *Network, s Signal) {

}

func processSignalTypeInvite(n *Network, s Signal) {

}

func processSignalTypeOffer(n *Network, s Signal) {

}

func processSignalTypeAnswer(n *Network, s Signal) {

}

func processSignalTypeConnectionSecret(n *Network, s Signal) {

}

func processSignalTypeConnectionProof(n *Network, s Signal) {

}

func processSignalTypeTrusted(n *Network, s Signal) {

}
