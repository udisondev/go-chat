package network

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"go-chat/config"
	"go-chat/handshake"
	"go-chat/middleware"
	"go-chat/pkg/closer"
	"go-chat/pkg/pack"
	"io"
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

func (n *Node) Attach(ctx context.Context, addr string, dispatcher func(hash []byte, inbox <-chan []byte) <-chan []byte) error {
	d := net.Dialer{}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	err = n.upgrade(ctx, c, dispatcher)
	if err != nil {
		c.Close()
		return err
	}

	return nil
}

func (n *Node) Listen(addr string, upgradeTimeout time.Duration, dispatcher func(hash []byte, inbox <-chan []byte) <-chan []byte) error {
	listenAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		return err
	}
	closer.Add(listener.Close)

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

				err := n.upgrade(ctx, c, dispatcher)
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

func (n *Node) upgrade(
	ctx context.Context,
	conn io.ReadWriteCloser,
	dispatch func(hash []byte, inbox <-chan []byte) <-chan []byte,
) error {
	h, err := handshake.With(ctx, conn, n.privkey.PublicKey(), n.pubsign)
	if err != nil {
		return fmt.Errorf("hanshake: %w", err)
	}

	inbox := make(chan []byte)
	go func() {
		defer close(inbox)

		buf := make([]byte, config.MaxInputLen)
		for {
			n, err := pack.ReadFrom(conn, buf)
			if err != nil {
				log.Printf("Error read pack: %v", err)
				return
			}
			tmp := make([]byte, n)
			copy(tmp, buf[:n])
			inbox <- tmp
		}
	}()

	wrapIn := middleware.ReadChecksum(inbox)
	wrapIn = middleware.ReadSignature(h.PubSign)(wrapIn)
	wrapIn = middleware.Decrypt(n.privkey, h.PubKey)(wrapIn)

	sum := sha256.Sum256(h.PubKey.Bytes())
	outbox := dispatch(sum[:], wrapIn)
	outbox = middleware.Encrypt(n.privkey, h.PubKey)(outbox)
	outbox = middleware.WriteSignature(n.privsign)(outbox)
	outbox = middleware.WriteChecksum(outbox)

	go func() {
		defer conn.Close()

		for out := range outbox {
			n, err := pack.WriteTo(conn, out)
			if err != nil {
				return
			}
			if n < len(out) {
				return
			}
		}
	}()

	return nil
}
