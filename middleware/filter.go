package middleware

import (
	"go-chat/config"
	"unsafe"
)

func Filter(isNew func(string) bool) func(<-chan []byte) <-chan []byte {
	return func(input <-chan []byte) <-chan []byte {
		output := make(chan []byte)

		go func() {
			defer close(output)

			for in := range input {
				if !isNew(unsafe.String(&in[0], config.NonceLen)) {
					continue
				}
				output <- in
			}
		}()

		return output
	}
}
