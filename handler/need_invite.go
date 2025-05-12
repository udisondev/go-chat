package handler

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"go-chat/model"
	"log"
	"time"
)

func (c *Connector) HandleNeedInvite(s model.Signal) {
	c.mu.Lock()
	defer c.mu.Unlock()

	signb, keyb := s.Payload[:ed25519.PublicKeySize], s.Payload[ed25519.PublicKeySize:]
	pubkey, err := ecdh.P256().NewPublicKey(keyb)
	if err != nil {
		log.Printf("handle need invite: parse pubkey: %v", err)
		return
	}
	ppubsign := ed25519.PublicKey(signb)

	privkey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		log.Printf("handle need invite: generate privkey: %v", err)
		return
	}
	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Printf("handle need invite: generate signature: %v", err)
		return
	}

	c.initq[s.RecipientString()] = &candidate{
		privkey:  privkey,
		privsign: privsign,
		pubsign:  pubsign,
		ppubkey:  pubkey,
		ppubsign: ppubsign,
	}

	go func() {
		<-time.After(time.Second * 10)
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.initq, s.RecipientString())
	}()

	out, err := model.NewSignal(
		model.SignalTypeReadyToInvite,
		c.node.Hash(),
		s.Author,
		append(pubsign, privkey.PublicKey().Bytes()...),
	)

	if err != nil {
		log.Printf("handle need invite: build signal: %v", err)
		return
	}

	c.dspch.Send(out)
}
