package network

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
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
	PubKey  *ecdh.PublicKey
	PubSign ed25519.PublicKey
}

type SignalType uint16

const (
	SignalNeedInvite SignalType = iota
	SignalNeedNewbieInvite
	SignalRedyToInviteNewbie
	SignalReadyToInvite
	SignalWaitOffer
	SignalWaitAnswer
	SignalAnswer
	SignalConnectionSecret
	SignalConnectionProof
	SignalTrusted
)

const (
	PubKeyLength           = 65
	PubSignLength          = 65
	SecretLength           = 26
	NonceLength            = 26
	RecipientLength        = 64
	AuthorLength           = 64
	ConnectionSecretLength = 26
	MinSignalLength        = NonceLength + RecipientLength + AuthorLength + 1
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
	if len(b) < MinSignalLength {
		return fmt.Errorf("too small signal len=%d", len(b))
	}

	pos := 0
	s.Nonce = string(b[pos : pos+NonceLength])
	pos += NonceLength

	s.Type = SignalType(b[pos])
	pos++

	if s.Type > SignalTrusted {
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
	out := make([]byte, MinSignalLength+len(s.Payload))
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
	if len(b) < ConnectionSecretLength+MinSignalLength {
		return errors.New("too small")
	}

	s.ConnectionSecret = string(b[:ConnectionSecretLength])
	err := s.Signal.Unmarshal(b[ConnectionSecretLength:])
	if err != nil {
		return err
	}

	return nil
}

func (s Handshake) Marshal() []byte {
	out := make([]byte, len(s.PubKey.Bytes())+len(s.PubSign))
	copy(out, s.PubKey.Bytes())
	copy(out[len(s.PubKey.Bytes()):], s.PubSign)

	return out
}

func (s *Handshake) Unmarshal(b []byte) error {
	if len(b) < PubKeyLength+PubSignLength {
		return errors.New("too small")
	}

	pubKey, err := ecdh.P256().NewPublicKey(b[:PubKeyLength])
	if err != nil {
		return err
	}
	s.PubKey = pubKey
	s.PubSign = ed25519.PublicKey(b[PubKeyLength:])

	return nil
}
