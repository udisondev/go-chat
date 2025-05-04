package hash

import (
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/hex"
)

func Peer(key *ecdh.PublicKey) string {
	sum := sha256.Sum256(key.Bytes())
	return string(hex.EncodeToString(sum[:]))
}
