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

type Peer struct {
	io.ReadWriteCloser
	hash []byte
}

func Attach(ctx context.Context, addr string) (*Peer, error) {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	closer.Add(conn.Close)
	if err != nil {
		return nil, err
	}

	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return UpgradeConn(ctx, key, pubsign, privsign, conn)
}

func UpgradeConn(
	ctx context.Context,
	key *ecdh.PrivateKey,
	pubsign ed25519.PublicKey,
	privsign ed25519.PrivateKey,
	rwc io.ReadWriteCloser,
) (*Peer, error) {
	h, err := handshake.With(ctx, rwc, key.PublicKey(), pubsign)
	if err != nil {
		return nil, err
	}
	rwc = middleware.Checksum(rwc)
	rwc = middleware.SignCheck(privsign, h.PubSign, rwc)
	rwc = middleware.Crypt(key, h.PubKey, rwc)

	sum := sha256.Sum256(h.PubKey.Bytes())

	return &Peer{
		ReadWriteCloser: rwc,
		hash:            sum[:],
	}, nil
}

func Listen(addr string, connTimeout time.Duration, h Handler) error {
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

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
				p, err := UpgradeConn(ctx, key, pubsign, privsign, c)
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

func (p *Peer) Hash() []byte {
	return p.hash
}
