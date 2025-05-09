package dispatcher

import (
	"crypto/rand"
	"go-chat/config"
	"unsafe"
)

type Signal []byte

type SignalType byte

const (
	NeedInvite SignalType = iota + 1
	Newbie
	RaiseYourHand
	ReadyToInvite
	WaitOffer
	WaitAnswer
	Answer
)

const (
	NonceLen       = config.NonceLen
	NonceStart     = 0
	AuthorLen      = 32
	AuthorStart    = NonceStart + NonceLen
	TypeLen        = 1
	TypeStart      = AuthorStart + AuthorLen
	RecipientLen   = 32
	RecipientStart = TypeStart + TypeLen
	PayloadStart   = RecipientStart + RecipientLen
	MinLen         = NonceLen + AuthorLen + RecipientLen
)

func NewSignal(t SignalType, author, recipient []byte, payload []byte) Signal {
	out := make([]byte, MinLen+len(payload)+1)
	rand.Read(out[:config.NonceLen])
	pos := config.NonceLen
	pos += copy(out[pos:], author)
	out[pos] = byte(t)
	pos++
	pos += copy(out[pos:], recipient)
	if payload != nil {
		copy(out[pos:], payload)
	}
	return Signal(out)
}

func (s Signal) Type() SignalType {
	return SignalType(s[TypeStart])
}

func (s Signal) Nonce() string {
	return unsafe.String(&s[0], NonceLen)
}

func (s Signal) Author() []byte {
	return s[AuthorStart:TypeStart]
}

func (s Signal) AuthorString() string {
	return unsafe.String(&s[AuthorStart], AuthorLen)
}

func (s Signal) Recipient() []byte {
	return s[RecipientStart:PayloadStart]
}

func (s Signal) RecipientString() string {
	return unsafe.String(&s[RecipientStart], RecipientLen)
}

func (s Signal) Payload() []byte {
	return s[PayloadStart:]
}
