package signal

import "unsafe"

type Outcome struct {
	Recipient string
	Resend    bool
	Payload   []byte
}

type SignalType byte

const (
	NeedInvite SignalType = iota + 1
	Newbie
	RaiseHand
	ReadyToInvite
	WaitOffer
	WaitAnswer
	Answer
)

const (
	AuthorLen      = 32
	AuthorStart    = 0
	TypeLen        = 1
	TypeStart      = AuthorStart + AuthorLen
	RecipientLen   = 32
	RecipientStart = TypeStart + TypeLen
	PayloadStart   = RecipientStart + RecipientLen
)

func BuildOutput(t SignalType, recipient []byte, payload []byte) Outcome {
	b := make([]byte, RecipientLen+len(payload)+1)
	if recipient == nil {
		recipient = make([]byte, RecipientLen)
	}
	pos := 0
	b[pos] = byte(t)
	pos++
	pos += copy(b[pos:], recipient)
	copy(b[pos:], payload)

	return b
}

func (s Income) Type() SignalType {
	return SignalType(s[TypeStart])
}

func (s Income) Author() []byte {
	return s[:AuthorLen]
}

func (s Income) Recipient() []byte {
	return s[RecipientStart:PayloadStart]
}

func (s Income) RecipientString() string {
	return unsafe.String(&s[RecipientStart], RecipientLen)
}

func (s Income) Payload() []byte {
	return s[PayloadStart:]
}

func (s Outcome) RecipientString() string {
	return unsafe.String(&s[RecipientStart], RecipientLen)
}
