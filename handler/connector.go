package handler

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"go-chat/dispatcher"
	"go-chat/model"
	"go-chat/network"
	"sync"
)

type Connector struct {
	mu       sync.Mutex
	hash     string
	node     *network.Node
	initq    map[string]*candidate
	respondq map[string]*candidate
	dspch    *dispatcher.Dispatcher
}

type candidate struct {
	privkey  *ecdh.PrivateKey
	privsign ed25519.PrivateKey
	pubsign  ed25519.PublicKey
	ppubkey  *ecdh.PublicKey
	ppubsign ed25519.PublicKey
}

func RunConnector(n *network.Node, d *dispatcher.Dispatcher) {
	c := &Connector{dspch: d, node: n}

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
}
