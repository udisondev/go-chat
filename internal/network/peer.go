package network

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"go-chat/pkg/crypt"
	"io"
	"sync"
)

type peer struct {
	state      peerState
	pubKey     *ecdh.PublicKey
	signature  ed25519.PublicKey
	mu         sync.Mutex
	disconnect func()
	rw         io.ReadWriter
}

type peerState uint8

const (
	peerInit peerState = iota
	peerConnected
	peerTrusted
	peerDisconnected
)

var (
	ErrPeerDisconnected = errors.New("disconnected")
	ErrSendingTimeout   = errors.New("sending timeout")
)

func newPeer(
	pubKey *ecdh.PublicKey,
	signature ed25519.PublicKey,
) *peer {
	p := peer{
		bufPool:   &sync.Pool{},
		pubKey:    pubKey,
		signature: signature,
	}

	p.disconnect = sync.OnceFunc(func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.state = peerDisconnected

		rwc.Close()
	})

	return &p
}

func (p *peer) Upgrade(s peerState) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.state = s
}

func (p *peer) Send(ctx context.Context, s Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == peerDisconnected {
		return ErrPeerDisconnected
	}

	sent := make(chan struct{})
	defer close(sent)

	errch := make(chan error)
	go func() {
		for written := 0; written < len(b); {
			n, err := p.rw.Write(b[written:])
			if err != nil {
				errch <- err
				return
			}
			written += n
		}
		sent <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ErrSendingTimeout
	case e := <-errch:
		return e
	case <-sent:
	}

	return nil
}

func (p *peer) Read(privateKey *ecdh.PrivateKey) {
	for {
		func() error {
			n.
		}()
	}
}
