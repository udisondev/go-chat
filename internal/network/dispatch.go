package network

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"go-chat/pkg/crypt"
	"sync"

	"github.com/pion/webrtc/v4"
)

var handlers = map[SignalType]func(*Network, Signal){
	SignalNeedNewbieInvite:   processNeedNewbieInvite,
	SignalRedyToInviteNewbie: processReadyToInviteNewbie,
	SignalReadyToInvite:      processReadyToInvite,
	SignalWaitOffer:          processWaitOffer,
	SignalWaitAnswer:         processWaitAnswer,
}

func (n *Network) dispatch(ctx context.Context, in <-chan Signal) {
	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-in:
			if !ok {
				return
			}
			h, ok := handlers[s.Type]
			if !ok {
				return
			}
			h(n, s)
		}
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

	toNewbie := newSignal(SignalReadyToInvite, peer.hash, n.hash, outh.Marshal())

	readyToInvite := ReadyToInviteNewbie{
		ConnectionSecret: rand.Text(),
		Signal:           toNewbie,
	}

	out := newSignal(SignalRedyToInviteNewbie, in.Author, n.hash, readyToInvite.Marshal())

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
	out := newSignal(SignalWaitOffer, in.Author, n.hash, payload)

	n.broadcast(out)
}

func processWaitOffer(n *Network, in Signal) {
	if n.hash != in.Recipient {
		n.broadcast(in)
		return
	}

	n.answererQueueeMu.Lock()
	defer n.answererQueueeMu.Unlock()
	answerer, ok := n.answererQueuee[in.Author]
	if !ok {
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

		outbox := n.interact(answerer.peer, inbox)
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

	encryptedPayload, err := crypt.Encrypt(SDP, n.privKey, answerer.peer.pubKey)
	if err != nil {
		return
	}

	signature := ed25519.Sign(n.privSign, encryptedPayload)
	encryptedPayload = append(encryptedPayload, signature...)

	answerer.pc = pc
	answerer.dc = dc

	n.broadcast(newSignal(SignalWaitAnswer, in.Author, n.hash, encryptedPayload))
}

func processWaitAnswer(n *Network, in Signal) {
	if n.hash != in.Recipient {
		n.broadcast(in)
		return
	}

	n.offererQueueeMu.Lock()
	defer n.offererQueueeMu.Unlock()

	offerer, ok := n.offererQueuee[in.Author]
	if !ok {
		return
	}

	payload, sign := crypt.SplitSignature(in.Payload)
	if !ed25519.Verify(offerer.peer.signature, payload, sign) {
		return
	}

	decryptedPayload, err := crypt.Decrypt(payload, n.privKey, offerer.peer.pubKey)
	if err != nil {
		return
	}

	var sd webrtc.SessionDescription
	err = json.Unmarshal(decryptedPayload, &sd)
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
	defer func() {
		if err != nil {
			pc.Close()
		}
	}()
	if err != nil {
		return
	}

	pc.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		dataChannel.OnOpen(func() {
			inbox := make(chan Signal)
			disconnect := sync.OnceFunc(func() {
				close(inbox)
			})

			outbox := n.interact(offerer.peer, inbox)
			go func() {
				defer disconnect()

				for s := range outbox {
					err := dataChannel.Send(s.Marshal())
					if err != nil {
						return
					}
				}
			}()

			dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
				if len(msg.Data) > maxSignalSize {
					disconnect()
					return
				}

				var s Signal
				s.Unmarshal(msg.Data)
				inbox <- s
			})

			dataChannel.OnClose(func() {
				disconnect()
			})
		})
	})

	err = pc.SetRemoteDescription(sd)
	if err != nil {
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)

	err = pc.SetLocalDescription(answer)
	if err != nil {
		return
	}

	<-gatherComplete
	anserSDP, err := json.Marshal(pc.LocalDescription())
	if err != nil {
		return
	}

	encryptedPayload, err := crypt.Encrypt(anserSDP, n.privKey, offerer.peer.pubKey)
	if err != nil {
		return
	}

	signature := ed25519.Sign(n.privSign, encryptedPayload)

	n.broadcast(newSignal(SignalAnswer, in.Author, n.hash, append(encryptedPayload, signature...)))
}

func processAnswer(n *Network, in Signal) {
	if n.hash != in.Recipient {
		n.broadcast(in)
		return
	}

	n.answererQueueeMu.Lock()
	defer n.answererQueueeMu.Unlock()

	answerer, ok := n.answererQueuee[in.Author]
	if !ok {
		return
	}

	payload, sign := crypt.SplitSignature(in.Payload)
	if !ed25519.Verify(answerer.peer.signature, payload, sign) {
		return
	}

	decryptedPayload, err := crypt.Decrypt(payload, n.privKey, answerer.peer.pubKey)
	if err != nil {
		return
	}

	var answer webrtc.SessionDescription
	err = json.Unmarshal(decryptedPayload, &answer)
	if err != nil {
		return
	}

	answerer.pc.SetRemoteDescription(answer)
}

func processSignalTypeConnectionSecret(n *Network, s Signal) {

}

func processSignalTypeConnectionProof(n *Network, s Signal) {

}

func processSignalTypeTrusted(n *Network, s Signal) {

}
