package network

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"go-chat/closer"
	"go-chat/handshake"
	"go-chat/middleware"
	"io"
	"log"
	"net"
	"time"
)

type Handler func(*Peer)

type Node struct {
	privkey  *ecdh.PrivateKey
	privsign ed25519.PrivateKey
	pubsign  ed25519.PublicKey
}

func NewNode() *Node {
	privkey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	return &Node{
		privkey:  privkey,
		privsign: privsign,
		pubsign:  pubsign,
	}
}

type Peer struct {
	io.ReadWriteCloser
	hash []byte
}

func (n *Node) Attach(ctx context.Context, addr string) (*Peer, error) {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	closer.Add(conn.Close)
	if err != nil {
		return nil, err
	}

	return n.NewPeer(ctx, conn)
}

func (n *Node) NewPeer(ctx context.Context, rwc io.ReadWriteCloser) (*Peer, error) {
	h, err := handshake.With(ctx, rwc, n.privkey.PublicKey(), n.pubsign)
	if err != nil {
		return nil, err
	}
	rwc = middleware.Checksum(rwc)
	rwc = middleware.SignCheck(n.privsign, h.PubSign, rwc)
	rwc = middleware.Crypt(n.privkey, h.PubKey, rwc)

	sum := sha256.Sum256(h.PubKey.Bytes())

	return &Peer{
		ReadWriteCloser: rwc,
		hash:            sum[:],
	}, nil
}

func (n *Node) Listen(addr string, connTimeout time.Duration, h Handler) error {
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
				ctx, cancel := context.WithTimeout(context.Background(), connTimeout)
				defer cancel()
				p, err := n.NewPeer(ctx, c)
				if err != nil {
					c.Close()
					return
				}
				h(p)
			}()
		}
	}()

	return nil
}

func (n *Node) Sign() (ed25519.PublicKey, ed25519.PrivateKey) {
	return n.pubsign, n.privsign
}

func (n *Node) ECDH() *ecdh.PrivateKey {
	return n.privkey
}

func (n *Node) Hash() []byte {
	sum := sha256.Sum256(n.privkey.PublicKey().Bytes())
	return sum[:]
}

func (p *Peer) Hash() []byte {
	return p.hash
}
