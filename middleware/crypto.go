package middleware

import (
	"crypto/ecdh"
	"errors"
	"fmt"
	"go-chat/config"
	"go-chat/pkg/crypto"
	"io"
	"log"
)

func Crypto(privKey *ecdh.PrivateKey, pubKey *ecdh.PublicKey) Middleware {
	return func(rw io.ReadWriter) io.ReadWriter {
		r, w := io.Pipe()

		go func() {
			defer r.Close()
			buf := make([]byte, config.MaxInputLen)
			for {
				err := readDownstream(buf, config.AESKeyLen, rw, w, func(b []byte) ([]byte, error) {
					out, err := crypto.Decrypt(b, privKey, pubKey)
					if err != nil {
						return nil, fmt.Errorf("decrypt message: %w", err)
					}
					return out, nil
				})
				if errors.Is(err, io.EOF) {
					return
				}
				if err != nil {
					log.Printf("Crypto: %v", err)
					return
				}
			}
		}()

		return &Wrapper{
			downstream: rw,
			Reader:     r,
			preparer: func(b []byte) ([]byte, error) {
				out, err := crypto.Encrypt(b, privKey, pubKey)
				if err != nil {
					return nil, fmt.Errorf("Crypt: encrypt message: %w", err)
				}
				return out, err
			},
		}
	}
}
