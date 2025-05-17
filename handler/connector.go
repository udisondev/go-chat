package handler

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"go-chat/dispatcher"
	"go-chat/model"
	"go-chat/network"
	"sync"
	"time"
)

type Connector struct {
	mu          sync.Mutex
	node        *network.Node
	initq       map[string]*candidate
	respondq    map[string]*candidate
	dspch       *dispatcher.Dispatcher
	connTimeout time.Duration
}

type candidate struct {
	pubkey  *ecdh.PublicKey
	pubsign ed25519.PublicKey
}

func RunConnector(n *network.Node, d *dispatcher.Dispatcher) *Connector {
	c := &Connector{
		node:        n,
		initq:       map[string]*candidate{},
		respondq:    map[string]*candidate{},
		dspch:       d,
		connTimeout: time.Second * 10,
	}

	go func() {
		for s := range d.Subscribe(model.SignalTypeNeedInvite) {
			c.HandleNeedInvite(s)
		}
	}()

	go func() {
		for s := range d.Subscribe(model.SignalTypeReadyToInvite) {
			c.HandleReadyToInvite(s)
		}
	}()

	go func() {
		for s := range d.Subscribe(model.SignalTypeWaitOffer) {
			c.HandleWaitOffer(s)
		}
	}()

	go func() {
		for s := range d.Subscribe(model.SignalTypeWaitAnswer) {
			c.HandleWaitAnswer(s)
		}
	}()

	return c
}
