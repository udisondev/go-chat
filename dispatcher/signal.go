package dispatcher

import "unsafe"

type Signal []byte

const (
	NeedInvite    byte = 0x00
	ReadyToInvite      = 0x01
	WaitOffer          = 0x02
	WaitAnswer         = 0x03
	Answer             = 0x04
)

const (
	TypeLen        = 1
	TypeStart      = 0
	AuthorLen      = 32
	AuthorStart    = TypeStart + TypeLen
	RecipientLen   = 32
	RecipientStart = AuthorStart + AuthorLen
	PayloadStart   = RecipientStart + RecipientLen
)

func (s Signal) Type() byte {
	return s[TypeStart]
}

func (s Signal) Author() []byte {
	return s[AuthorStart:RecipientStart]
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
