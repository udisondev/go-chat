//go:generate go-enum .
package model

import (
	"crypto/rand"
	"errors"
	"unsafe"
)

type Signal []byte

// ENUM(
// NeedConnect
// Offer
// Answer
// Candidate
// )
type SignalType uint8

const (
	TypeLen  = 1
	KeyLen   = 16
	NonceLen = 16

	TypeStart    = 0
	KeyStart     = TypeStart + TypeLen
	NonceStart   = KeyStart + KeyLen
	PayloadStart = NonceStart + NonceLen

	MinLen = TypeLen + KeyLen + NonceLen
)

func FormatSignal(b []byte) (Signal, error) {
	if len(b) < MinLen {
		return nil, errors.New("too short")
	}
	return Signal(b), nil
}

func NewSignal(t SignalType, key []byte, payload []byte) (Signal, error) {
	if len(key) != KeyLen {
		return nil, errors.New("invalid key len")
	}

	out := make([]byte, MinLen+len(payload))
	out[0] = byte(t)
	pos := 1

	pos += copy(out[pos:], key)

	rand.Read(out[pos : pos+NonceLen])
	pos += NonceLen

	copy(out[pos:], payload)

	return Signal(out), nil
}

func (s Signal) Type() SignalType {
	return SignalType(s[TypeStart])
}

func (s Signal) Key() []byte {
	return s[KeyStart:NonceStart]
}

func (s Signal) KeyString() string {
	return unsafe.String(&s[KeyStart], KeyLen)
}

func (s Signal) NonceString() string {
	return unsafe.String(&s[NonceStart], NonceLen)
}

func (s Signal) Payload() []byte {
	return s[PayloadStart:]
}

func GenerateKey() []byte {
	key := make([]byte, KeyLen)
	rand.Read(key)
	return key
}
