package handler

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"go-chat/dispatcher"
	"go-chat/model"
	"go-chat/network"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type rwcadap struct {
	io.Reader
	io.Writer
	close func() error
}

func Test_HandleNeedInvite(t *testing.T) {
	d := dispatcher.New()
	n := network.NewNode()

	c := RunConnector(n, d)
	c.connTimeout = time.Second

	hash, rwc, r, w := genRwc()
	d.Dispatch(hash, &rwc)

	author := make([]byte, 32)
	rand.Read(author)
	privKey, _ := ecdh.P256().GenerateKey(rand.Reader)
	pubsign, _, _ := ed25519.GenerateKey(rand.Reader)
	payload := append(pubsign, privKey.PublicKey().Bytes()...)
	s, _ := model.NewSignal(model.SignalTypeNeedInvite, author, hash, payload)
	sb := []byte(s)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		expectedKey := n.ECDH().PublicKey()
		expectedSign, _ := n.Sign()
		buf := make([]byte, 1024)
		l, err := r.Read(buf)
		assert.NoError(t, err)
		s, err := model.FormatSignal(buf[:l])
		assert.NoError(t, err)
		signb, keyb := s.Payload()[:ed25519.PublicKeySize], s.Payload()[ed25519.PublicKeySize:]
		ppubkey, err := ecdh.P256().NewPublicKey(keyb)
		assert.NoError(t, err)
		ppubsign := ed25519.PublicKey(signb)
		assert.Equal(t, expectedKey, ppubkey)
		assert.Equal(t, expectedSign, ppubsign)
		assert.Equal(t, model.SignalTypeReadyToInvite, s.Type())
	}()

	w.Write(sb)
	wg.Wait()
	peer, ok := c.initq[s.RecipientString()]
	assert.True(t, ok)
	assert.Equal(t, privKey.PublicKey(), peer.pubkey)
	assert.Equal(t, pubsign, peer.pubsign)
	<-time.After(c.connTimeout + time.Second)
	_, ok = c.initq[s.RecipientString()]
	assert.False(t, ok)
}

func (r *rwcadap) Close() error {
	if r.close == nil {
		return nil
	}
	return r.close()
}

func genRwc() ([]byte, rwcadap, io.Reader, io.Writer) {
	ph := make([]byte, 32)
	rand.Read(ph)
	pr, w := io.Pipe()
	r, pw := io.Pipe()
	return ph, rwcadap{Reader: pr, Writer: pw}, r, w
}
