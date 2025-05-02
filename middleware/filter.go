package middleware

import (
	"errors"
	"go-chat/config"
	"io"
	"log"
	"unsafe"
)

var errIsDuplicate = errors.New("duplicate")

func Filter(exists func(string) bool) Middleware {
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
					return b[config.NonceLen:], nil
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
		}
	}
}
