package network

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"fmt"
	"go-chat/config"
	"go-chat/handshake"
	"go-chat/hash"
	"go-chat/middleware"
	"go-chat/pkg/pack"
	"io"
	"log"
)

type Upgrader struct {
	privkey  *ecdh.PrivateKey
	privsign ed25519.PrivateKey
	pubsign  ed25519.PublicKey
}

func (u *Upgrader) Upgrade(
	ctx context.Context,
	conn io.ReadWriteCloser,
	dispatch func(hash string, inbox <-chan []byte) <-chan []byte,
) error {
	h, err := handshake.With(ctx, conn, u.privkey.PublicKey(), u.pubsign)
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
	wrapIn = middleware.Decrypt(u.privkey, h.PubKey)(wrapIn)

	outbox := dispatch(hash.PubKeyToString(h.PubKey), wrapIn)
	outbox = middleware.Encrypt(u.privkey, h.PubKey)(outbox)
	outbox = middleware.WriteSignature(u.privsign)(outbox)
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
