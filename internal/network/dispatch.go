package network

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"go-chat/pkg/crypt"
	"log/slog"
	"sync"

	"github.com/pion/webrtc/v4"
)

var handlers = map[SignalType]func(*Network, Signal){
	SignalNeedNewbieInvite:   processNeedNewbieInvite,
	SignalRedyToInviteNewbie: processReadyToInviteNewbie,
	SignalReadyToInvite:      processReadyToInvite,
	SignalWaitOffer:          processWaitOffer,
	SignalWaitAnswer:         processWaitAnswer,
	SignalAnswer:             processAnswer,
	SignalConnectionSecret:   processConnectionSecret,
	SignalConnectionProof:    processConnectionProof,
	SignalTrusted:            processTrusted,
}

func (n *Network) dispatch(ctx context.Context, in <-chan Signal) {
	log := n.logger.With("method", "dispatch")
	for {
		select {
		case <-ctx.Done():
			log.Info("Context closed")
			return
		case s, ok := <-in:
			if !ok {
				return
			}

			log.Debug("Received", slog.String("signal", s.Type.String()))
			h, ok := handlers[s.Type]
			if !ok {
				log.Debug("Has not suitable handler")
				return
			}
			h(n, s)
		}
	}
}

func processNeedNewbieInvite(n *Network, in Signal) {
	log := n.logger.With(
		"method",
		"processNeedNewbieInvite",
		"author",
		in.Author,
	)

	if n.freeSlots.Add(1) > maxFreeSlotsCount {
		log.Debug("Has no free slot")
		n.freeSlots.Add(-1)
		go n.broadcast(in)
		return
	}

	var inh Handshake
	err := inh.Unmarshal(in.Payload)
	if err != nil {
		log.Error("Unmarshal handshake", slog.Any("err", err))
		return
	}

	n.initianorsMu.Lock()
	defer n.initianorsMu.Unlock()

	peer := n.newPeer(inh.PubKey, inh.PubSign)
	connectionProof := rand.Text()
	n.initiatorQueuee[peer.hash] = &initiator{
		peer:            peer,
		connectionProof: connectionProof,
	}

	log = log.With("newbie", peer.hash)

	outh := Handshake{
		PubKey:  n.privKey.PublicKey(),
		PubSign: n.pubSignature,
	}

	toNewbie := newSignal(SignalReadyToInvite, peer.hash, n.hash, outh.Marshal())

	readyToInvite := ReadyToInviteNewbie{
		ConnectionProof: connectionProof,
		Signal:          toNewbie,
	}

	out := newSignal(SignalRedyToInviteNewbie, in.Author, n.hash, readyToInvite.Marshal())

	go n.broadcast(out)

	log.Info("ReadyToInviteNewbie was sent")
}

func processReadyToInviteNewbie(n *Network, in Signal) {
	log := n.logger.With(
		"method",
		"processReadyToInviteNewbie",
		"author",
		in.Author,
	)

	if in.Recipient != n.hash {
		log.Debug("There is not for me")
		go n.broadcast(in)
		return
	}

	var rinv ReadyToInviteNewbie
	err := rinv.Unmarshal(in.Payload)
	if err != nil {
		log.Error("Unmarshal ReadyToInviteNewbie", slog.Any("err", err))
		return
	}

	log = log.With("newbie", rinv.Recipient)

	n.respondersMu.RLock()
	defer n.respondersMu.RUnlock()
	inviteWaiter, ok := n.onboarding[rinv.Signal.Recipient]
	if !ok {
		log.Warn("Unknown newbie")
		return
	}

	inviteWaiter.mu.Lock()
	defer inviteWaiter.mu.Unlock()
	if len(inviteWaiter.secrets) > reqConns {
		log.Warn("Has enougth connection candidates")
		return
	}

	inviteWaiter.secrets[rinv.Author] = rinv.ConnectionProof

	inviteWaiter.peer.send(rinv.Signal)
	log.Info("Invite was sent")
}

func processReadyToInvite(n *Network, in Signal) {
	log := n.logger.With(
		"method",
		"processReadyToInvite",
		"author",
		in.Author,
	)

	if in.Recipient != n.hash {
		log.Debug("There is not for me")
		go n.broadcast(in)
		return
	}

	var h Handshake
	err := h.Unmarshal(in.Payload)
	if err != nil {
		log.Error("Unmarshal handshake", slog.Any("err", err))
		return
	}

	expectedSecret := rand.Text()

	n.respondersMu.Lock()
	defer n.respondersMu.Unlock()

	peer := n.newPeer(h.PubKey, h.PubSign)
	n.respondersQueuee[peer.hash] = &responder{
		peer: peer,
	}

	encryptedSecret, err := crypt.Encrypt([]byte(expectedSecret), n.privKey, h.PubKey)
	if err != nil {
		log.Error("Enctypt ", slog.Any("err", err))
		return
	}

	signature := ed25519.Sign(n.privSign, encryptedSecret)
	payload := append(encryptedSecret, signature...)
	out := newSignal(SignalWaitOffer, in.Author, n.hash, payload)

	go n.broadcast(out)

	log.Info("Wait offer was sent")
}

func processWaitOffer(n *Network, in Signal) {
	log := n.logger.With(
		"method",
		"processWaitOffer",
		"author",
		in.Author,
	)

	if n.hash != in.Recipient {
		log.Debug("There is not for me")
		go n.broadcast(in)
		return
	}

	n.initianorsMu.Lock()
	defer n.initianorsMu.Unlock()
	initiator, ok := n.initiatorQueuee[in.Author]
	if !ok {
		log.Warn("Unknown initiator")
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
		log.Error("Create new peer connection", slog.Any("err", err))
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
		log.Error("Create datachannel", slog.Any("err", err))
		return
	}
	defer func() {
		if err == nil {
			return
		}
		dc.Close()
	}()

	dc.OnOpen(func() {
		dc.Send(
			newSignal(
				SignalConnectionSecret,
				initiator.peer.hash,
				n.hash,
				[]byte(initiator.connectionProof),
			).Marshal())
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Error("Create offer", slog.Any("err", err))
		return
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		log.Error("Set local description", slog.Any("err", err))
		return
	}

	SDP, err := json.Marshal(pc.LocalDescription())
	if err != nil {
		log.Error("Marshal local description", slog.Any("err", err))
		return
	}

	encryptedPayload, err := crypt.Encrypt(SDP, n.privKey, initiator.peer.pubKey)
	if err != nil {
		log.Error("Encrypt SDP", slog.Any("err", err))
		return
	}

	signature := ed25519.Sign(n.privSign, encryptedPayload)
	encryptedPayload = append(encryptedPayload, signature...)

	initiator.pc = pc
	initiator.dc = dc

	go n.broadcast(newSignal(SignalWaitAnswer, in.Author, n.hash, encryptedPayload))

	log.Info("Wait answer was sent")
}

func processWaitAnswer(n *Network, in Signal) {
	log := n.logger.With(
		"method",
		"processWaitAnswer",
		"author",
		in.Author,
	)

	if n.hash != in.Recipient {
		log.Debug("There is not for me")
		go n.broadcast(in)
		return
	}

	n.respondersMu.Lock()
	defer n.respondersMu.Unlock()

	responder, ok := n.respondersQueuee[in.Author]
	if !ok {
		log.Warn("Unknown reponder")
		return
	}

	payload, sign := crypt.SplitSignature(in.Payload)
	if !ed25519.Verify(responder.peer.signature, payload, sign) {
		log.Warn("Invalid sign")
		return
	}

	decryptedPayload, err := crypt.Decrypt(payload, n.privKey, responder.peer.pubKey)
	if err != nil {
		log.Error("Decrypt SDP", slog.Any("err", err))
		return
	}

	var sd webrtc.SessionDescription
	err = json.Unmarshal(decryptedPayload, &sd)
	if err != nil {
		log.Error("Unmarshal SDP", slog.Any("err", err))
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
		log.Error("Create new peer connection", slog.Any("err", err))
		return
	}
	defer func() {
		if err != nil {
			pc.Close()
		}
	}()

	pc.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		dataChannel.OnOpen(func() {
			log := n.logger.With("peer", responder.peer.hash)
			inbox := make(chan Signal)
			disconnect := sync.OnceFunc(func() {
				close(inbox)
			})

			outbox := n.interact(responder.peer, inbox)
			go func() {
				defer disconnect()

				log.Debug("Outbox is ready")
				for s := range outbox {
					err := dataChannel.Send(s.Marshal())
					if err != nil {
						return
					}
				}
			}()

			log.Debug("Inbox is ready")
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
		log.Error("Set remote description", slog.Any("err", err))
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Error("Create answer", slog.Any("err", err))
		return
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)

	err = pc.SetLocalDescription(answer)
	if err != nil {
		log.Error("Set local description", slog.Any("err", err))
		return
	}

	<-gatherComplete
	anserSDP, err := json.Marshal(pc.LocalDescription())
	if err != nil {
		log.Error("Marshal local description", slog.Any("err", err))
		return
	}

	encryptedPayload, err := crypt.Encrypt(anserSDP, n.privKey, responder.peer.pubKey)
	if err != nil {
		log.Error("Encrypt SDP", slog.Any("err", err))
		return
	}

	signature := ed25519.Sign(n.privSign, encryptedPayload)

	go n.broadcast(newSignal(SignalAnswer, in.Author, n.hash, append(encryptedPayload, signature...)))
	log.Info("Answer was sent")
}

func processAnswer(n *Network, in Signal) {
	log := n.logger.With(
		"method",
		"processAnswer",
		"author",
		in.Author,
	)

	if n.hash != in.Recipient {
		log.Debug("There is not for me")
		go n.broadcast(in)
		return
	}

	n.initianorsMu.Lock()
	defer n.initianorsMu.Unlock()
	initiator, ok := n.initiatorQueuee[in.Author]
	if !ok {
		log.Warn("Unknown initiator")
		return
	}

	payload, sign := crypt.SplitSignature(in.Payload)
	if !ed25519.Verify(initiator.peer.signature, payload, sign) {
		log.Warn("Invalid sign")
		return
	}

	decryptedPayload, err := crypt.Decrypt(payload, n.privKey, initiator.peer.pubKey)
	if err != nil {
		log.Error("Dectypr SDP", slog.Any("err", err))
		return
	}

	var answer webrtc.SessionDescription
	err = json.Unmarshal(decryptedPayload, &answer)
	if err != nil {
		log.Error("Unmarshal SDP", slog.Any("err", err))
		return
	}

	initiator.pc.SetRemoteDescription(answer)
}

func processConnectionSecret(n *Network, in Signal) {
	log := n.logger.With(
		"method",
		"processConnectionSecret",
		"author",
		in.Author,
	)
	if n.hash != in.Recipient {
		log.Debug("There is not for me")
		return
	}

	n.respondersMu.Lock()
	defer n.respondersMu.Unlock()
	responder, ok := n.respondersQueuee[in.Author]
	if !ok {
		log.Warn("Unknown responder")
		return
	}

	n.peersMu.Lock()
	defer n.peersMu.Unlock()
	n.peers[in.Author] = responder.peer

	delete(n.respondersQueuee, in.Author)

	go n.broadcast(newSignal(SignalConnectionProof, "", n.hash, append(in.Payload, []byte(in.Author)...)))
	log.Info("Connection proof was sent")
}

func processConnectionProof(n *Network, in Signal) {
	log := n.logger.With(
		"method",
		"processSignalTypeConnectionProof",
		"author",
		in.Author,
	)
	n.onboardingMu.Lock()
	defer n.onboardingMu.Unlock()
	newbie, ok := n.onboarding[in.Author]
	if !ok {
		log.Warn("Unknown newbie")
		return
	}

	newbie.mu.Lock()
	defer newbie.mu.Unlock()

	secret, connector := string(in.Payload[:26]), string(in.Payload[26:])
	expectedSecret, ok := newbie.secrets[connector]
	if !ok {
		log.Warn("Unknown connection proof")
		return
	}

	if secret != expectedSecret {
		log.Warn("Different secrets")
		return
	}

	newbie.connections++

	if newbie.connections < reqConns {
		log.Info("Need more connection proofs")
		return
	}

	go n.broadcast(newSignal(SignalTrusted, "", n.hash, []byte(in.Author)))
	log.Info("Trusted was sent")

	n.peersMu.Lock()
	defer n.peersMu.Unlock()
	n.peers[newbie.peer.hash] = newbie.peer
	delete(n.onboarding, newbie.peer.hash)
}

func processTrusted(n *Network, in Signal) {
	log := n.logger.With(
		"method",
		"processSignalTypeTrusted",
		"author",
		in.Author,
	)
	trusted := string(in.Payload)
	n.initianorsMu.Lock()
	defer n.initianorsMu.Unlock()
	initiator, ok := n.initiatorQueuee[trusted]
	if !ok {
		log.Warn("Unknown initiator")
		go n.broadcast(in)
		return
	}

	n.peersMu.Lock()
	defer n.peersMu.Unlock()
	n.peers[initiator.peer.hash] = initiator.peer

	delete(n.initiatorQueuee, trusted)

	dc := initiator.dc

	inbox := make(chan Signal)
	disconnect := sync.OnceFunc(func() {
		close(inbox)
	})

	outbox := n.interact(initiator.peer, inbox)
	go func() {
		log := n.logger.With("peer", initiator.peer.hash)
		defer disconnect()

		log.Debug("Oubox is ready")
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

	log.Debug("Inbox is ready")

	dc.OnClose(func() {
		disconnect()
	})

	log.Info("Trust")
}
