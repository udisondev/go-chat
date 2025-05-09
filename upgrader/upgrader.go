package upgrade

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"go-chat/config"
	"go-chat/handshake"
	"go-chat/middleware"
	"go-chat/pkg/pack"
	"io"
	"log"
	"time"
)

type dispatcher interface {
	Dispatch(hash []byte, inbox <-chan []byte) (<-chan []byte, error)
}

type Upgrader struct {
	privkey  *ecdh.PrivateKey
	privsign ed25519.PrivateKey
	pubsign  ed25519.PublicKey
	timeout  time.Duration
	d        dispatcher
}

func New(timeout time.Duration, d dispatcher) *Upgrader {
	privkey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return &Upgrader{
		privkey:  privkey,
		privsign: privsign,
		pubsign:  pubsign,
		timeout:  timeout,
		d:        d,
	}
}

func (u *Upgrader) Upgrade(
	conn io.ReadWriteCloser,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), u.timeout)
	defer cancel()

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

	sum := sha256.Sum256(h.PubKey.Bytes())
	outbox, err := u.d.Dispatch(sum[:], wrapIn)
	if err != nil {
		return err
	}

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
