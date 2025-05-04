package test

import (
	"context"
	"go-chat/network"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Connect(t *testing.T) {
	addr := "127.0.0.1:7998"
	hello := "Hello server"
	wg := sync.WaitGroup{}
	wg.Add(1)
	network.Listen(addr, time.Second*5, func(_ string, inbox <-chan []byte) <-chan []byte {
		outbox := make(chan []byte)
		go func() {
			defer wg.Done()
			for in := range inbox {
				assert.Equal(t, hello, string(in))
				outbox <- in
				return
			}
		}()
		return outbox
	})

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*5)
	defer cancel()

	err := network.Attach(ctx, addr, func(_ string, inbox <-chan []byte) <-chan []byte {
		outbox := make(chan []byte, 1)
		outbox <- []byte(hello)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for in := range inbox {
				assert.Equal(t, hello, string(in))
				return
			}
		}()
		return outbox
	})

	assert.NoError(t, err)
	wg.Wait()
}
