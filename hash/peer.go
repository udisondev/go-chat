package hash

import (
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/hex"
)

func PubKeyToString(key *ecdh.PublicKey) string {
	return string(hex.EncodeToString(PubKey(key)))
}

func PubKey(key *ecdh.PublicKey) []byte {
	sum := sha256.Sum256(key.Bytes())
	return sum[:]
}
