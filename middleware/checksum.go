package middleware

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"go-chat/config"
	"io"
	"log"
)

func Checksum(rw io.ReadWriter) io.ReadWriter {
	r, w := io.Pipe()

	go func() {
		defer r.Close()
		buf := make([]byte, config.MaxInputLen)
		for {
			err := readDownstream(buf, sha256.Size, rw, w, func(b []byte) ([]byte, error) {
				payload := b[sha256.Size:]
				givenSum, actualSum := b[:sha256.Size], sha256.Sum256(payload)
				if !bytes.Equal(givenSum, actualSum[:]) {
					return nil, errors.New("checksum failed")
				}
				return payload, nil
			})
			if err != nil {
				log.Printf("Checksum: %v", err)
				return
			}
		}
	}()

	return &Wrapper{
		downstream: rw,
		Reader:     r,
		preparer: func(b []byte) ([]byte, error) {
			sum := sha256.Sum256(b)
			return append(sum[:], b...), nil
		},
	}
}
