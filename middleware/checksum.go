package middleware

import (
	"bytes"
	"crypto/sha256"
	"log"
)

func ReadChecksum(input <-chan []byte) <-chan []byte {
	output := make(chan []byte)

	go func() {
		defer close(output)

		for in := range input {
			checksum, payload := in[:sha256.Size], in[sha256.Size:]
			actual := sha256.Sum256(payload)
			if !bytes.Equal(checksum, actual[:]) {
				log.Println("Checksum failed")
				return
			}
			output <- payload
		}
	}()

	return output
}

func WriteChecksum(input <-chan []byte) <-chan []byte {

	output := make(chan []byte)

	go func() {
		defer close(output)

		for in := range input {
			sum := sha256.Sum256(in)
			output <- append(sum[:], in...)
		}
	}()

	return output
}
