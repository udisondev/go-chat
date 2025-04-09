package network

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

type Network struct {
	hash string

	privKey *ecdh.PrivateKey

	privSignature ed25519.PrivateKey
	pubSignature  ed25519.PublicKey

	freeSlots atomic.Int32

	peersMu sync.RWMutex
	peers   map[string]*peer

	onboardingMu sync.RWMutex
	onboarding   map[string]*newbie

	queueeMu sync.RWMutex
	queuee   map[string]*peer

	cache *cache
}

type newbie struct {
	mu      sync.Mutex
	secrets map[string]string
	peer    *peer
}

func (n *Network) interact(p *peer) error {
	if p.state != peerConnected {
		return errors.New("invalid state")
	}

	if p.rw == nil {
		return errors.New("peer.rw is nil")
	}

	go func() {
		defer p.disconnect()

		n.dispatch(n.filter(read(p.rw)))
	}()

	return nil
}

func read(r io.Reader) <-chan Signal {
	out := make(chan Signal)

	go func() {
		defer close(out)

		for {
			var mlen uint16
			err := binary.Read(r, binary.BigEndian, &mlen)
			if err != nil {
				return
			}

			buf := make([]byte, mlen)

			for read := 0; read < int(mlen); {
				n, err := r.Read(buf[read:])
				if err != nil {
					return
				}
				read += n
			}

			var s Signal
			err = s.UnmarshalBinary(buf)
			if err != nil {
				return
			}

			out <- s
		}

	}()

	return out
}

func (n *Network) filter(in <-chan Signal) <-chan Signal {
	out := make(chan Signal)

	go func() {
		defer close(out)

		for s := range in {
			if !n.cache.putIfAbsent(s.Nonce) {
				continue
			}

			out <- s
		}
	}()

	return out
}

func (n *Network) broadcast(s Signal) {

}
