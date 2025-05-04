package middleware

import (
	"crypto/ecdh"
	"go-chat/pkg/crypto"
	"log"
)

func Decrypt(privkey *ecdh.PrivateKey, pubkey *ecdh.PublicKey) func(<-chan []byte) <-chan []byte {
	return func(input <-chan []byte) <-chan []byte {
		output := make(chan []byte)

		go func() {
			defer close(output)

			for in := range input {
				decrypted, err := crypto.Decrypt(in, privkey, pubkey)
				if err != nil {
					log.Println("Error decrypt message: %w", err)
				}
				output <- decrypted
			}
		}()

		return output
	}
}

func Encrypt(privkey *ecdh.PrivateKey, pubkey *ecdh.PublicKey) func(<-chan []byte) <-chan []byte {
	return func(input <-chan []byte) <-chan []byte {
		output := make(chan []byte)

		go func() {
			defer close(output)

			for in := range input {
				encrypted, err := crypto.Encrypt(in, privkey, pubkey)
				if err != nil {
					log.Println("Error encrypt message")
					return
				}
				output <- encrypted
			}
		}()

		return output
	}
}
