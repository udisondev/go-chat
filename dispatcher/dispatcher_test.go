package dispatcher

import (
	"bytes"
	"crypto/rand"
	"go-chat/model"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type rwcadap struct {
	io.Reader
	io.Writer
}

func Test_Dispatch(t *testing.T) {
	t.Run("check peers count", func(t *testing.T) {
		d := New()
		hash := []byte(rand.Text())
		d.Dispatch(hash, &rwcadap{Reader: new(bytes.Buffer)})

		assert.Len(t, d.peers, 1)
	})

	t.Run("check disconnect", func(t *testing.T) {
		d := New()
		hash := []byte(rand.Text())
		d.Dispatch(hash, &rwcadap{Reader: new(bytes.Buffer)})

		assert.Len(t, d.peers, 1)
		d.Disconnect(hash)
		<-time.After(time.Second)
		assert.Len(t, d.peers, 0)
	})

	t.Run(
		"should publish",
		func(t *testing.T) {
			d := New()
			hash := []byte(rand.Text())
			r, w := io.Pipe()
			rwc := &rwcadap{Reader: r}
			d.Dispatch(hash, rwc)
			randPayload := make([]byte, 12)
			rand.Read(randPayload)
			expected, _ := model.NewSignal(
				model.SignalTypeNeedInvite,
				make([]byte, 32),
				make([]byte, 32),
				randPayload,
			)
			ch := d.SubscribeType(model.SignalTypeNeedInvite)
			assert.Len(t, d.typesubs, 1)

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()

				msg := <-ch
				assert.Equal(t, expected, msg)
			}()

			expb := []byte(expected)
			w.Write(expb)
			wg.Wait()
		},
	)

	t.Run("Send to peer", func(t *testing.T) {
		d := New()
		hash := []byte(rand.Text())
		r, w := io.Pipe()
		rwc := &rwcadap{Reader: new(bytes.Buffer), Writer: w}
		d.Dispatch(hash, rwc)
		randPayload := make([]byte, 12)
		expected, _ := model.NewSignal(
			model.SignalTypeNeedInvite,
			make([]byte, 32),
			make([]byte, 32),
			randPayload,
		)
		d.Send(expected)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			buf := make([]byte, 1024)
			n, err := r.Read(buf)
			assert.NoError(t, err)

			s, err := model.FormatSignal(buf[:n])
			assert.NoError(t, err)
			assert.Equal(t, expected, s)
		}()
		wg.Wait()
	})
}

func (r *rwcadap) Close() error {
	return nil
}
