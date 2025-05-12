package network

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"go-chat/netcrypt"
	"go-chat/pack"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type rwcadapter struct {
	io.Reader
	io.Writer
}

func Test_NewPeer(t *testing.T) {
	n := NewNode()
	dr, w := io.Pipe()
	r, dw := io.Pipe()
	io.Pipe()
	rwc := rwcadapter{Reader: dr, Writer: dw}
	pprivkey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	ppubsign, pprivsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	go func() {
		payload := append(ppubsign, pprivkey.PublicKey().Bytes()...)
		w.Write(payload)
	}()

	go func() {
		b := make([]byte, 200)
		r.Read(b)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	p, err := n.NewPeer(ctx, &rwc)
	defer p.Close()

	assert.NoError(t, err)
	assert.NotNil(t, p)

	t.Run("Write to peer", func(t *testing.T) {
		text := rand.Text()

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			buf := make([]byte, 1024)
			l, err := pack.ReadFrom(r, buf)
			payload := buf[:l]
			checksum, payload := payload[:32], payload[32:]
			actualChecksum := sha256.Sum256(payload)
			assert.Equal(t, actualChecksum[:], checksum)
			sign, payload := payload[:ed25519.SignatureSize], payload[ed25519.SignatureSize:]
			assert.True(t, ed25519.Verify(n.pubsign, payload, sign))
			decrypted, err := netcrypt.Decrypt(payload, pprivkey, n.privkey.PublicKey())
			assert.NoError(t, err)
			assert.Equal(t, []byte(text), decrypted)

		}()
		p.Write([]byte(text))
		wg.Wait()
	})

	t.Run("Read from peer", func(t *testing.T) {
		text := rand.Text()
		expected := []byte(text)
		expected, _ = netcrypt.Encrypt(expected, pprivkey, n.privkey.PublicKey())
		expected = append(ed25519.Sign(pprivsign, expected), expected...)
		sum := sha256.Sum256(expected)
		expected = append(sum[:], expected...)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			buf := make([]byte, 1024)
			n, err := p.Read(buf)
			assert.NoError(t, err)
			assert.Equal(t, []byte(text), buf[:n])
		}()
		pack.WriteTo(w, expected)
		wg.Wait()
	})
}

func (r *rwcadapter) Close() error {
	return nil
}
