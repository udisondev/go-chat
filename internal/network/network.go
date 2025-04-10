package network

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"go-chat/pkg/closer"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/pion/webrtc/v4"
)

type (
	Network struct {
		logger       *slog.Logger
		hash         string
		privKey      *ecdh.PrivateKey
		privSign     ed25519.PrivateKey
		pubSignature ed25519.PublicKey

		freeSlots atomic.Int32
		peersMu   sync.RWMutex
		peers     map[string]*peer

		onboardingMu sync.RWMutex
		onboarding   map[string]*newbie

		respondersMu     sync.RWMutex
		respondersQueuee map[string]*responder
		initianorsMu     sync.RWMutex
		initiatorQueuee  map[string]*initiator

		inbox chan Signal

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
		mu          sync.Mutex
		secrets     map[string]string
		connections int
		peer        *peer
	}

	responder struct {
		peer *peer
		pc   *webrtc.PeerConnection
		dc   *webrtc.DataChannel
	}

	initiator struct {
		peer            *peer
		connectionProof string
		pc              *webrtc.PeerConnection
		dc              *webrtc.DataChannel
	}
)

func (n *Network) Run(ctx context.Context) {
	inbox := make(chan Signal)
	defer close(inbox)

	wg := sync.WaitGroup{}
	workerCtx, stop := context.WithCancel(ctx)
	closer.Add(func() error {
		stop()
		return nil
	})

	for range runtime.NumCPU() {
		wg.Add(1)

		go func() {
			defer wg.Done()
			n.dispatch(workerCtx, inbox)
		}()
	}

	wg.Wait()
}

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
	log := n.logger.With("peer", hash)

	disconnected := atomic.Bool{}
	mu := sync.RWMutex{}
	disconnect := sync.OnceFunc(func() {
		mu.Lock()
		defer mu.Unlock()

		if !disconnected.CompareAndSwap(false, true) {
			log.Warn("Already disconnected!")
			return
		}

		n.freeSlots.Add(-1)

		n.peersMu.Lock()
		defer n.peersMu.Unlock()

		delete(n.peers, hash)
		close(outbox)
	})
	p.disconnect = disconnect

	send := func(s Signal) bool {
		mu.RLock()
		if disconnected.Load() {
			mu.RUnlock()
			return false
		}

		select {
		case outbox <- s:
			mu.RUnlock()
			return true
		default:
			mu.RUnlock()
			disconnect()
		}

		return false
	}
	p.send = send

	go func() {
		defer p.disconnect()

		for s := range n.filter(inbox) {
			n.inbox <- s
		}
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

func (n *Network) broadcast(s Signal) {
	n.peersMu.RLock()
	defer n.peersMu.Unlock()

	for _, p := range n.peers {
		go p.send(s)
	}
}
