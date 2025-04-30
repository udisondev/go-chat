package dispatcher

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"sync"
)

type Initiator struct {
	mu     sync.Mutex
	queuee map[string]offerer
}

type offerer struct {
	pubSign ed25519.PublicKey
	pubKey  *ecdh.PublicKey
}

func (i *Initiator) handleReadyToInvite(s Signal) Signal { return nil }
