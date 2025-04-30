package handshake

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"io"
)

func With(
	ctx context.Context,
	rw io.ReadWriter,
	pubkey *ecdh.PublicKey,
	pubsign ed25519.PublicKey,
) (*ecdh.PublicKey, ed25519.PublicKey, error) {
	return nil, nil, nil
}
