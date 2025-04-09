package network

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type Signal struct {
	Nonce     string
	Type      SignalType
	Recipient string
	Author    string
	Payload   []byte
}

type ReadyToInviteNewbie struct {
	ConnectionSecret string
	Signal
}

type Handshake struct {
	PublicKey *ecdh.PublicKey
	Signature ed25519.PublicKey
}

type Offer struct {
	Secret []byte
	SDP    []byte
}

type SignalType uint16

const (
	SignalTypeNeedInvite SignalType = iota
	SignalTypeNeedNewbieInvite
	SignalTypeRedyToInviteNewbie
	SignalTypeReadyToInvite
	SignalTypeWaitOffer
	SignalTypeSendInvite
	SignalTypeInvite
	SignalTypeOffer
	SignalTypeAnswer
	SignalTypeConnectionSecret
	SignalTypeConnectionProof
	SignalTypeTrusted
)

const (
	NonceLength            = 26
	RecipientLength        = 64
	AuthorLength           = 64
	ConnectionSecretLength = 26
	MinLengh               = NonceLength + RecipientLength + AuthorLength + 1
)

func newSignal(
	t SignalType,
	recipient string,
	author string,
	payload []byte,
) Signal {
	return Signal{
		Nonce:     rand.Text(),
		Type:      t,
		Recipient: recipient,
		Author:    author,
		Payload:   payload,
	}
}

func (s *Signal) Unmarshal(b []byte) error {
	if len(b) < MinLengh {
		return fmt.Errorf("too small signal len=%d", len(b))
	}

	pos := 0
	s.Nonce = string(b[pos : pos+NonceLength])
	pos += NonceLength

	s.Type = SignalType(b[pos])
	pos++

	if s.Type > SignalTypeTrusted {
		return fmt.Errorf("unknown signal type: %d", s.Type)
	}

	s.Author = string(b[pos : pos+AuthorLength])
	pos += AuthorLength
	s.Recipient = string(b[pos : pos+RecipientLength])
	pos += RecipientLength
	s.Payload = b[pos:]

	return nil
}

func (s Signal) Marshal() []byte {
	out := make([]byte, MinLengh+len(s.Payload))
	pos := 0
	pos += copy(out[pos:], []byte(s.Nonce))
	out[pos] = byte(s.Type)
	pos++
	pos += copy(out[pos:], []byte(s.Author))
	pos += copy(out[pos:], []byte(s.Recipient))
	copy(out[pos:], s.Payload)

	return out
}

func (s ReadyToInviteNewbie) Marshal() []byte {
	return append([]byte(s.ConnectionSecret), s.Signal.Marshal()...)
}

func (s *ReadyToInviteNewbie) Unmarshal(b []byte) error {
	s.ConnectionSecret = string(b[:ConnectionSecretLength])
	err := s.Signal.Unmarshal(b[ConnectionSecretLength:])
	if err != nil {
		return err
	}

	return nil
}

func (s Handshake) Marshal() []byte {
	out := make([]byte, len(s.PublicKey.Bytes())+len(s.Signature))
	copy(out, s.PublicKey.Bytes())
	copy(out[len(s.PublicKey.Bytes()):], s.Signature)

	return out
}

func (s *Handshake) Unmarshal(b []byte) error {
	pubKey, err := ecdh.P256().NewPublicKey(b[:65])
	if err != nil {
		return err
	}
	s.PublicKey = pubKey
	s.Signature = ed25519.PublicKey(b[65:])

	return nil
}
