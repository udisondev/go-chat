package network

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"go-chat/handshake"
	"go-chat/middleware"
	"go-chat/pkg/closer"
	"log"
	"net"
	"time"
)

func Attach(ctx context.Context, addr string, mws ...middleware.Middleware) (*Peer, error) {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	p, err := upgradeConn(ctx, conn, mws...)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return p, nil
}

func Listen(
	addr string,
	upgradeTimeout time.Duration,
	dispatch func(*Peer),
	mws ...middleware.Middleware,
) error {
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
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("accept conn: %v", err)
				continue
			}
			go func() {
				ctx, close := context.WithTimeout(context.Background(), upgradeTimeout)
				defer close()

				p, err := upgradeConn(ctx, conn, mws...)
				if err != nil {
					conn.Close()
					log.Printf("interact with new conn: %v", err)
					return
				}
				dispatch(p)
			}()
		}
	}()

	return nil
}

func upgradeConn(ctx context.Context, conn net.Conn, mws ...middleware.Middleware) (*Peer, error) {
	privkey, privsign, pubsign, err := generateKeys()
	if err != nil {
		return nil, fmt.Errorf("generate keys: %w", err)
	}

	ppubkey, ppubsign, err := handshake.With(ctx, conn, privkey.PublicKey(), pubsign)
	if err != nil {
		return nil, fmt.Errorf("handshake error: %w", err)
	}

	reqMids := []middleware.Middleware{
		middleware.Checksum,
		middleware.Signature(privsign, ppubsign),
		middleware.Crypto(privkey, ppubkey),
	}

	reqMids = append(reqMids, mws...)

	return NewPeer(
		string(ppubkey.Bytes()),
		conn,
		reqMids...,
	), nil
}

func generateKeys() (*ecdh.PrivateKey, ed25519.PrivateKey, ed25519.PublicKey, error) {
	privkey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	return privkey, privsign, pubsign, nil
}
