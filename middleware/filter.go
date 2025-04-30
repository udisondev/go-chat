package middleware

import (
	"crypto/rand"
	"errors"
	"go-chat/config"
	"io"
	"log"
	"unsafe"
)

var errIsDuplicate = errors.New("duplicate")

func Filter(put func(string), exists func(string) bool) Middleware {
	return func(rw io.ReadWriter) io.ReadWriter {
		r, w := io.Pipe()

		go func() {
			defer r.Close()
			buf := make([]byte, config.MaxInputLen)
			for {
				err := readDownstream(buf, config.NonceLen, rw, w, func(b []byte) ([]byte, error) {
					nonce := unsafe.String(&b[0], config.NonceLen)
					if exists(nonce) {
						return nil, errIsDuplicate
					}
					return b, nil
				})
				if errors.Is(err, errIsDuplicate) {
					continue
				}
				if err != nil {
					log.Printf("Filter: %v", err)
					return
				}
			}
		}()

		return &Wrapper{
			downstream: rw,
			Reader:     r,
			preparer: func(b []byte) ([]byte, error) {
				nonce := make([]byte, config.NonceLen)
				rand.Read(nonce)
				put(unsafe.String(&nonce[0], config.NonceLen))
				return append(nonce, b...), nil
			},
		}
	}
}
