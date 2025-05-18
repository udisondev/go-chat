package handler

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"go-chat/model"
	"log"
	"time"
)

func (c *Connector) HandleNeedInvite(s model.Signal) {
	c.mu.Lock()
	defer c.mu.Unlock()

	signb, keyb := s.Payload()[:ed25519.PublicKeySize], s.Payload()[ed25519.PublicKeySize:]
	ppubkey, err := ecdh.P256().NewPublicKey(keyb)
	if err != nil {
		log.Printf("handle need invite: parse pubkey: %v", err)
		return
	}
	ppubsign := ed25519.PublicKey(signb)

	c.initq[s.RecipientString()] = &candidate{
		pubkey:  ppubkey,
		pubsign: ppubsign,
	}

	go func() {
		<-time.After(c.connTimeout)
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.initq, s.RecipientString())
	}()

	pubsign, _ := c.node.Sign()
	pubkey := c.node.ECDH().PublicKey()

	out, err := model.NewSignal(
		model.SignalTypeReadyToInvite,
		s.Key(),
		c.node.Hash(),
		s.Author(),
		append(pubsign, pubkey.Bytes()...),
	)

	if err != nil {
		log.Printf("handle need invite: build signal: %v", err)
		return
	}

	c.dspch.Send(out)
}
