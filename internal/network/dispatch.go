package network

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go-chat/pkg/crypt"
	"sync"

	"github.com/pion/webrtc/v4"
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

	expectedSecret := rand.Text()

	n.answererQueueeMu.Lock()
	defer n.answererQueueeMu.Unlock()

	peer := n.newPeer(inh.PubKey, inh.PubSign)
	n.answererQueuee[peer.hash] = &answerer{
		peer:           peer,
		expectedSecret: expectedSecret,
	}

	outh := Handshake{
		PubKey:  n.privKey.PublicKey(),
		PubSign: n.pubSignature,
	}

	toNewbie := newSignal(SignalTypeReadyToInvite, peer.hash, n.hash, outh.Marshal())

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

	n.offererQueueeMu.RLock()
	defer n.offererQueueeMu.RUnlock()
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

	inviteWaiter.peer.send(rinv.Signal)
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

	expectedSecret := rand.Text()

	n.offererQueueeMu.Lock()
	defer n.offererQueueeMu.Unlock()

	peer := n.newPeer(h.PubKey, h.PubSign)
	n.offererQueuee[peer.hash] = &offerer{
		peer:           peer,
		expectedSecret: expectedSecret,
	}

	encryptedSecret, err := crypt.Encrypt([]byte(expectedSecret), n.privKey, h.PubKey)
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

	n.answererQueueeMu.Lock()
	defer n.answererQueueeMu.Unlock()
	answerWaiter, ok := n.answererQueuee[in.Author]
	if !ok {
		return
	}

	secret, senderSignature := crypt.SplitSignature(in.Payload)
	if !ed25519.Verify(answerWaiter.peer.signature, secret, senderSignature) {
		return
	}

	decryptedSecret, err := crypt.Decrypt(secret, n.privKey, answerWaiter.peer.pubKey)
	if err != nil {
		return
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun.l.google.com:19302",
				},
			},
		},
	}
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return
	}

	defer func() {
		if err == nil {
			return
		}
		pc.Close()
	}()

	dc, err := pc.CreateDataChannel("network", nil)
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			return
		}
		dc.Close()
	}()

	dc.OnOpen(func() {
		inbox := make(chan Signal)
		disconnect := sync.OnceFunc(func() {
			close(inbox)
		})

		outbox := n.interact(answerWaiter.peer, inbox)
		go func() {
			defer disconnect()

			for s := range outbox {
				err := dc.Send(s.Marshal())
				if err != nil {
					return
				}
			}
		}()

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if len(msg.Data) > maxSignalSize {
				disconnect()
				return
			}

			var s Signal
			s.Unmarshal(msg.Data)
			inbox <- s
		})

		dc.OnClose(func() {
			disconnect()
		})
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		return
	}

	SDP, err := json.Marshal(pc.LocalDescription())
	if err != nil {
		return
	}

	of := Offer{
		Secret:       answerWaiter.expectedSecret,
		SolvedSecret: string(decryptedSecret),
		SDP:          SDP,
	}

	encryptedPayload, err := crypt.Encrypt(of.Marshal(), n.privKey, answerWaiter.peer.pubKey)
	if err != nil {
		return
	}

	signature := ed25519.Sign(n.privSign, encryptedPayload)
	encryptedPayload = append(encryptedPayload, signature...)

	n.broadcast(newSignal(SignalTypeOffer, in.Author, n.hash, encryptedPayload))
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
