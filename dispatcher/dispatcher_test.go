package dispatcher

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_EntrypointPutRaisYourHand(t *testing.T) {
	myhash := make([]byte, 32)
	rand.Read(myhash)
	peerhash := make([]byte, 32)
	rand.Read(peerhash)
	d := NewServer(myhash)
	ch := make(chan []byte)
	defer close(ch)
	outbox := d.Dispatch(peerhash, ch)
	msg := <-outbox
	s := Signal(msg)
	assert.Equal(t, RaiseYourHand, s.Type())
}
