package handler

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"go-chat/dispatcher"
	"sync"
	"time"
)

type Connector struct {
	mu       sync.Mutex
	count    int
	maxCount int
	queuee   map[string]*initiator
}

type initiator struct {
	pubkey  *ecdh.PublicKey
	pubsign ed25519.PublicKey
}

func (c *Connector) HandleNeedInvite(d *dispatcher.Distapcher, s dispatcher.Signal) (dispatcher.Signal, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	if c.count > c.maxCount {
		return s, nil
	}

	init := s.AuthorString()

	psb, pkb := s.Payload()[:ed25519.PublicKeySize], s.Payload()[ed25519.PublicKeySize:]
	pubkey, err := ecdh.P256().NewPublicKey(pkb)
	if err != nil {
		return nil, nil
	}

	pubsign := ed25519.PublicKey(psb)

	c.queuee[init] = &initiator{
		pubkey:  pubkey,
		pubsign: pubsign,
	}

	go func() {
		<-time.After(time.Second * 10)
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.queuee, init)
	}()

}
