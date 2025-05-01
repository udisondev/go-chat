package handshake

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_With(t *testing.T) {
	type adapter struct {
		io.Reader
		io.Writer
	}

	t.Run("success", func(t *testing.T) {
		r, w := io.Pipe()
		peerPubSign, _, err := ed25519.GenerateKey(rand.Reader)
		assert.NoError(t, err)
		peerPrivKey, err := ecdh.P256().GenerateKey(rand.Reader)
		assert.NoError(t, err)

		pubSign, _, err := ed25519.GenerateKey(rand.Reader)
		assert.NoError(t, err)
		privKey, err := ecdh.P256().GenerateKey(rand.Reader)
		assert.NoError(t, err)

		peerPayload := make([]byte, len(peerPrivKey.PublicKey().Bytes())+ed25519.PublicKeySize)
		copy(peerPayload[:ed25519.PublicKeySize], peerPubSign)
		copy(peerPayload[ed25519.PublicKeySize:], peerPrivKey.PublicKey().Bytes())
		buf := new(bytes.Buffer)
		buf.Write(peerPayload)
		a := adapter{
			Writer: w,
			Reader: buf,
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			in := make([]byte, len(peerPayload))
			for read := 0; read < len(in); {
				n, err := r.Read(in[read:])
				if errors.Is(err, io.EOF) {
					break
				}
				assert.NoError(t, err)
				read += n
			}
			expected := make([]byte, len(in))
			copy(expected[:ed25519.PublicKeySize], pubSign)
			copy(expected[ed25519.PublicKeySize:], privKey.PublicKey().Bytes())
			assert.Equal(t, expected, in)
		}()

		actualPeerPubKey, actualPeerPubSign, err := With(t.Context(), a, privKey.PublicKey(), pubSign)
		assert.NoError(t, err)
		assert.Equal(t, peerPrivKey.PublicKey(), actualPeerPubKey)
		assert.Equal(t, peerPubSign, actualPeerPubSign)
		wg.Wait()
	})

	t.Run("invalid pubkey", func(t *testing.T) {
		pubSign, _, err := ed25519.GenerateKey(rand.Reader)
		assert.NoError(t, err)
		privKey, err := ecdh.P256().GenerateKey(rand.Reader)
		assert.NoError(t, err)

		peerPubSign, _, err := ed25519.GenerateKey(rand.Reader)
		assert.NoError(t, err)
		peerPrivKey := make([]byte, len(privKey.PublicKey().Bytes()))
		rand.Read(peerPrivKey)

		peerPayload := make([]byte, len(peerPrivKey)+ed25519.PublicKeySize)
		copy(peerPayload[:ed25519.PublicKeySize], peerPubSign)
		copy(peerPayload[ed25519.PublicKeySize:], peerPrivKey)
		buf := new(bytes.Buffer)
		buf.Write(peerPayload)
		a := adapter{
			Writer: new(bytes.Buffer),
			Reader: buf,
		}
		_, _, err = With(t.Context(), a, privKey.PublicKey(), pubSign)
		assert.ErrorContains(t, err, "parse public key")
	})

	t.Run("context closed", func(t *testing.T) {
		pubSign, _, err := ed25519.GenerateKey(rand.Reader)
		assert.NoError(t, err)
		privKey, err := ecdh.P256().GenerateKey(rand.Reader)
		assert.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err = With(ctx, new(bytes.Buffer), privKey.PublicKey(), pubSign)
		assert.ErrorContains(t, err, "context closed")
	})
}
