package conn

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"go-chat/config"
	"go-chat/handshake"
	"go-chat/hash"
	"go-chat/middleware"
	pack "go-chat/pkg/packlen"
	"io"
	"log"
)

type Middleware func(<-chan []byte) <-chan []byte
type Dispatcher func(string, <-chan []byte) <-chan []byte

func Upgrade(
	ctx context.Context,
	conn io.ReadWriteCloser,
	dspch func(string, <-chan []byte) <-chan []byte,
) error {
	privkey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate ecdh: %w", err)
	}

	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate sign: %w", err)
	}

	ppubkey, ppubsign, err := handshake.With(ctx, conn, privkey.PublicKey(), pubsign)
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
	wrapIn = middleware.ReadSignature(ppubsign)(wrapIn)
	wrapIn = middleware.Decrypt(privkey, ppubkey)(wrapIn)

	outbox := dspch(hash.Peer(ppubkey), wrapIn)

	outMws := []Middleware{
		middleware.Encrypt(privkey, ppubkey),
		middleware.WriteSignature(privsign),
		middleware.WriteChecksum,
	}

	for _, mw := range outMws {
		outbox = mw(outbox)
	}

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
