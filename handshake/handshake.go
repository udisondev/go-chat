package handshake

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
)

type Handshake struct {
	PubKey  *ecdh.PublicKey
	PubSign ed25519.PublicKey
}

func With(
	ctx context.Context,
	rw io.ReadWriter,
	pubkey *ecdh.PublicKey,
	pubsign ed25519.PublicKey,
) (Handshake, error) {
	input := make(chan []byte)
	errCh := make(chan error)
	defer close(errCh)

	go func() {
		payload := append(pubsign, pubkey.Bytes()...)
		for written := 0; written < len(payload); {
			n, err := rw.Write(payload[written:])
			if err != nil {
				errCh <- err
				return
			}
			written += n
		}
	}()

	go func() {
		defer close(input)
		payload := make([]byte, len(pubkey.Bytes())+ed25519.PublicKeySize)
		for read := 0; read < len(payload); {
			n, err := rw.Read(payload[read:])
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				errCh <- err
				return
			}
			read += n
		}
		input <- payload
	}()

	select {
	case <-ctx.Done():
		return Handshake{}, errors.New("context closed")
	case e := <-errCh:
		return Handshake{}, e
	case b := <-input:
		sigBytes, keyBytes := b[:ed25519.PublicKeySize], b[ed25519.PublicKeySize:]
		peerPubKey, err := ecdh.P256().NewPublicKey(keyBytes)
		if err != nil {
			return Handshake{}, fmt.Errorf("parse public key: %w", err)
		}
		peerPubSign := ed25519.PublicKey(sigBytes)
		return Handshake{
			PubKey:  peerPubKey,
			PubSign: peerPubSign,
		}, nil
	}
}
