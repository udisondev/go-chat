package handshake

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
)

func With(
	ctx context.Context,
	rw io.ReadWriter,
	pubkey *ecdh.PublicKey,
	pubsign ed25519.PublicKey,
) (*ecdh.PublicKey, ed25519.PublicKey, error) {
	input := make(chan []byte)
	defer close(input)

	errCh := make(chan error)
	defer close(errCh)

	go func() {
		for written := 0; written < len(pubsign); {
			n, err := rw.Write(pubsign[written:])
			if err != nil {
				errCh <- err
				return
			}
			written += n
		}
		for written := 0; written < len(pubkey.Bytes()); {
			n, err := rw.Write(pubkey.Bytes()[written:])
			if err != nil {
				errCh <- err
				return
			}
			written += n
		}
	}()

	go func() {
		payload := make([]byte, len(pubkey.Bytes())+ed25519.PublicKeySize)
		for read := 0; read < len(input); {
			n, err := rw.Read(payload[read:])
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
		return nil, nil, errors.New("context closed")
	case e := <-errCh:
		return nil, nil, e
	case b := <-input:
		sigBytes, keyBytes := b[:ed25519.PublicKeySize], b[ed25519.PublicKeySize:]
		peerPubKey, err := ecdh.P256().NewPublicKey(keyBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse public key: %w", err)
		}
		peerPubSign := ed25519.PublicKey(sigBytes)
		return peerPubKey, peerPubSign, nil
	}
}
