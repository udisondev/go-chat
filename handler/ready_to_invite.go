package handler

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"go-chat/model"
	"log"
	"time"
)

func (c *Connector) HandleReadyToInvite(s model.Signal) {
	c.mu.Lock()
	defer c.mu.Unlock()

	signb, keyb := s.Payload()[:ed25519.PublicKeySize], s.Payload()[ed25519.PublicKeySize:]
	ppubkey, err := ecdh.P256().NewPublicKey(keyb)
	if err != nil {
		log.Printf("handle need invite: parse pubkey: %v", err)
		return
	}
	ppubsign := ed25519.PublicKey(signb)

	c.respondq[s.RecipientString()] = &candidate{
		pubkey:  ppubkey,
		pubsign: ppubsign,
	}

	go func() {
		<-time.After(c.connTimeout)
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.respondq, s.RecipientString())
	}()
}
