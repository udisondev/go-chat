//go:generate go-enum .
package model

import (
	"errors"
	"unsafe"
)

type Signal struct {
	Payload   []byte
	Type      SignalType
	Recipient []byte
	Author    []byte
}

// ENUM(
// NeedInnvite
// ReadyToInvite
// WaitOffer
// WaitAnswer
// )
type SignalType uint8

func FormatSignal(b []byte) (Signal, error) {
	return Signal{}, nil
}

func NewSignal(t SignalType, author, recipient, payload []byte) (Signal, error) {
	if len(author) != 32 {
		return Signal{}, errors.New("author is required")
	}
	if len(recipient) != 32 {
		return Signal{}, errors.New("recipient is required")
	}
	return Signal{}, nil
}

func (s Signal) Marshal() ([]byte, error) {
	return nil, nil
}

func (s *Signal) Unmarshal(b []byte) error {
	return nil
}

func (s *Signal) RecipientString() string {
	return unsafe.String(&s.Recipient[0], len(s.Recipient))
}
