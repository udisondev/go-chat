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
	return Signal{
		Payload:   payload,
		Type:      t,
		Recipient: recipient,
		Author:    author,
	}, nil
}

func (s Signal) Marshal() ([]byte, error) {
	totalLen := 1 + 64 + len(s.Payload)
	out := make([]byte, totalLen)
	out[0] = byte(s.Type)
	pos := 1
	pos += copy(out[pos:], s.Author)
	pos += copy(out[pos:], s.Recipient)
	copy(out[pos:], s.Payload)

	return out, nil
}

func (s *Signal) Unmarshal(b []byte) error {
	if len(b) < 65 {
		return errors.New("too short input")
	}
	pos := 0
	s.Type = SignalType(b[pos])
	pos++
	s.Author = b[pos : pos+32]
	pos += 32
	s.Recipient = b[pos : pos+32]
	pos += 32
	s.Payload = b[pos:]

	return nil
}

func (s *Signal) RecipientString() string {
	return unsafe.String(&s.Recipient[0], len(s.Recipient))
}
