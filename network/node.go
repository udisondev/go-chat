package network

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"go-chat/pkg/closer"
	"log"
	"net"
	"time"
)

type Node struct {
	privkey  *ecdh.PrivateKey
	privsign ed25519.PrivateKey
	pubsign  ed25519.PublicKey
}

func NewNode(
	privkey *ecdh.PrivateKey,
	privsign ed25519.PrivateKey,
	pubsign ed25519.PublicKey,
) *Node {
	return &Node{
		privkey:  privkey,
		privsign: privsign,
		pubsign:  pubsign,
	}
}

type Dispatcher interface {
	Dispatch(hash []byte, inbox <-chan []byte) <-chan []byte
}

func (n *Node) Attach(ctx context.Context, addr string, dispatcher Dispatcher) error {
	d := net.Dialer{}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	u := Upgrader{
		privkey:  n.privkey,
		privsign: n.privsign,
		pubsign:  n.pubsign,
	}

	err = u.Upgrade(ctx, c, dispatcher.Dispatch)
	if err != nil {
		c.Close()
		return err
	}

	return nil
}

func (n *Node) Listen(addr string, upgradeTimeout time.Duration, d Dispatcher) error {
	listenAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		return err
	}
	closer.Add(listener.Close)

	u := Upgrader{
		privkey:  n.privkey,
		privsign: n.privsign,
		pubsign:  n.pubsign,
	}

	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				log.Printf("accept conn: %v", err)
				continue
			}
			go func() {
				ctx, close := context.WithTimeout(context.Background(), upgradeTimeout)
				defer close()

				err := u.Upgrade(ctx, c, d.Dispatch)
				if err != nil {
					c.Close()
					log.Printf("interact with new conn: %v", err)
					return
				}
			}()
		}
	}()

	return nil
}
