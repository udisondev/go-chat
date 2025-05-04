package test

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
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
	entrypoint, err := genNode()
	assert.NoError(t, err)
	entrypoint.Listen(addr, time.Second*5, func(_ []byte, inbox <-chan []byte) <-chan []byte {
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

	attached, err := genNode()
	assert.NoError(t, err)
	err = attached.Attach(ctx, addr, func(_ []byte, inbox <-chan []byte) <-chan []byte {
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

func genNode() (*network.Node, error) {
	privkey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil
	}
	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil
	}
	n := network.NewNode(privkey, privsign, pubsign)
	return n, nil
}
