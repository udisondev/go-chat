package network

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
)

type (
	Network struct {
		hash string

		privKey *ecdh.PrivateKey

		privSign     ed25519.PrivateKey
		pubSignature ed25519.PublicKey

		freeSlots atomic.Int32

		peersMu sync.RWMutex
		peers   map[string]*peer

		onboardingMu sync.RWMutex
		onboarding   map[string]*newbie

		offererQueueeMu sync.RWMutex
		offererQueuee   map[string]*offerer

		answererQueueeMu sync.RWMutex
		answererQueuee   map[string]*answerer

		cache *cache
	}

	peer struct {
		hash       string
		pubKey     *ecdh.PublicKey
		signature  ed25519.PublicKey
		disconnect func()
		send       func(Signal) bool
	}

	newbie struct {
		mu      sync.Mutex
		secrets map[string]string
		peer    *peer
	}

	offerer struct {
		peer           *peer
		expectedSecret string
	}

	answerer struct {
		peer           *peer
		expectedSecret string
	}
)

func (n *Network) newPeer(
	pubKey *ecdh.PublicKey,
	signature ed25519.PublicKey,
) *peer {
	sum := sha256.Sum256(pubKey.Bytes())
	return &peer{
		hash:      hex.EncodeToString(sum[:]),
		pubKey:    pubKey,
		signature: signature,
	}

}

func (n *Network) interact(p *peer, inbox <-chan Signal) <-chan Signal {
	outbox := make(chan Signal, 256)

	sum := sha256.Sum256(p.pubKey.Bytes())
	hash := hex.EncodeToString(sum[:])

	disconnected := atomic.Bool{}
	disconnect := func() {
		if !disconnected.CompareAndSwap(false, true) {
			return
		}

		n.peersMu.Lock()
		defer n.peersMu.Unlock()

		delete(n.peers, hash)
		close(outbox)
	}
	p.disconnect = disconnect

	send := func(s Signal) bool {
		if disconnected.Load() {
			return false
		}

		select {
		case outbox <- s:
			return true
		default:
			disconnect()
		}

		return false
	}
	p.send = send

	go func() {
		defer p.disconnect()

		n.dispatch(n.filter(inbox))
	}()

	return outbox
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

func (n *Network) broadcast(s Signal, excludes ...string) {

}
