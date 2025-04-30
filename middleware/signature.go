package middleware

import (
	"crypto/ed25519"
	"errors"
	"go-chat/config"
	"io"
	"log"
)

func Signature(
	privateSign ed25519.PrivateKey,
	publicSign ed25519.PublicKey,
) Middleware {
	return func(rw io.ReadWriter) io.ReadWriter {
		r, w := io.Pipe()

		go func() {
			defer r.Close()
			buf := make([]byte, config.MaxInputLen)
			for {
				err := readDownstream(buf, ed25519.SignatureSize, rw, w, func(b []byte) ([]byte, error) {
					signature, payload := b[:ed25519.SignatureSize], b[ed25519.SignatureSize:]
					if !ed25519.Verify(publicSign, payload, signature) {
						return nil, errors.New("invalid signature")
					}
					return payload, nil
				})
				if err != nil {
					log.Printf("Signature: %v", err)
					return
				}
			}
		}()

		return &Wrapper{
			downstream: rw,
			Reader:     r,
			preparer: func(b []byte) ([]byte, error) {
				signature := ed25519.Sign(privateSign, b)
				return append(signature, b...), nil
			},
		}
	}
}
