package middleware

import (
	"crypto/ed25519"
	"log"
)

func ReadSignature(pubsign ed25519.PublicKey) func(<-chan []byte) <-chan []byte {
	return func(input <-chan []byte) <-chan []byte {
		output := make(chan []byte)

		go func() {
			defer close(output)

			for in := range input {
				signature, payload := in[:ed25519.SignatureSize], in[ed25519.SignatureSize:]
				if !ed25519.Verify(pubsign, payload, signature) {
					log.Println("Signature failed")
					return
				}

				output <- payload
			}
		}()

		return output
	}
}

func WriteSignature(privsign ed25519.PrivateKey) func(<-chan []byte) <-chan []byte {
	return func(input <-chan []byte) <-chan []byte {
		output := make(chan []byte)

		go func() {
			defer close(output)

			for in := range input {
				output <- append(ed25519.Sign(privsign, in), in...)
			}
		}()

		return output
	}
}
