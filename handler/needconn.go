package handler

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"go-chat/model"
	wrtc "go-chat/webrtc"
	"log"
)

func NeedConn(s model.Signal, hasFree func() bool, send func(model.Signal)) {
	if !hasFree() {
		send(s)
		return
	}
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		log.Println("NeedConn: ecdh.GenerateKey:", err)
		return
	}

	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Println("NeedConn: ed25519.GenerateKey:", err)
		return
	}

	pc, err := wrtc.Setup()
	if err != nil {
		log.Println("NeedConn: wrtc.Setup:", err)
		return
	}
	sdp, err := wrtc.BuildOffer(pc)
	if err != nil {
		log.Println("NeedConn: wrtc.BuildOffer:", err)
		return
	}

	payload := make([]byte, len(sdp)+len(pubsign)+len(key.PublicKey().Bytes()), ed25519.SignatureSize)
	pos := ed25519.SignatureSize
	pos += copy(payload[pos:], pubsign)
	pos += copy(payload[pos:], key.PublicKey().Bytes())
	pos += copy(payload[pos:], sdp)
	signature := ed25519.Sign(privsign, payload)
	copy(payload[0:ed25519.SignatureSize], signature)

	mkey := model.GenerateKey()
	signal, err := model.NewSignal(
		model.SignalTypeOffer,
		mkey,
		payload,
	)
	if err != nil {
		log.Println("NeedConn: model.NewSignal:", err)
		return
	}

	send(signal)
}
