package middleware

import (
	"errors"
	"go-chat/config"
	"io"
	"log"
	"unsafe"
)

var errIsDuplicate = errors.New("duplicate")

func Filter(isNew func(string) bool, rw io.ReadWriter) io.ReadWriter {
	r, w := io.Pipe()

	go func() {
		defer r.Close()
		buf := make([]byte, config.MaxInputLen)
		for {
			err := readDownstream(buf[:0], config.NonceLen, rw, w, func(b []byte) ([]byte, error) {
				nonce := unsafe.String(&b[0], 12)
				if !isNew(nonce) {
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
			return b, nil
		},
	}
}
