package dispatcher

import (
	"bytes"
	"crypto/rand"
	"go-chat/model"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type rwcadap struct {
	io.Reader
	io.Writer
	close func() error
}

func Test_Dispatch(t *testing.T) {
	// t.Run("check peers count", func(t *testing.T) {
	// 	d := New()
	// 	hash := []byte(rand.Text())
	// 	d.Dispatch(hash, &rwcadap{Reader: new(bytes.Buffer)})

	// 	assert.Len(t, d.peers, 1)
	// })

	// t.Run("check disconnect", func(t *testing.T) {
	// 	t.Parallel()
	// 	d := New()
	// 	hash := []byte(rand.Text())
	// 	d.Dispatch(hash, &rwcadap{Reader: new(bytes.Buffer)})

	// 	assert.Len(t, d.peers, 1)
	// 	d.Disconnect(hash)
	// 	<-time.After(time.Second)
	// 	assert.Len(t, d.peers, 0)
	// })

	// t.Run(
	// 	"should publish",
	// 	func(t *testing.T) {
	// 		d := New()
	// 		hash := []byte(rand.Text())
	// 		r, w := io.Pipe()
	// 		rwc := &rwcadap{Reader: r}
	// 		d.Dispatch(hash, rwc)
	// 		randPayload := make([]byte, 12)
	// 		rand.Read(randPayload)
	// 		expected, _ := model.NewSignal(
	// 			model.SignalTypeNeedInvite,
	// 			make([]byte, 32),
	// 			make([]byte, 32),
	// 			randPayload,
	// 		)
	// 		ch := d.Subscribe(model.SignalTypeNeedInvite)
	// 		assert.Len(t, d.topics, 1)

	// 		wg := sync.WaitGroup{}
	// 		wg.Add(1)
	// 		go func() {
	// 			defer wg.Done()

	// 			msg := <-ch
	// 			assert.Equal(t, expected, msg)
	// 		}()

	// 		expb, err := expected.Marshal()
	// 		assert.NoError(t, err)
	// 		w.Write(expb)
	// 		wg.Wait()
	// 	},
	// )

	// t.Run("Send to peer", func(t *testing.T) {
	// 	d := New()
	// 	hash := []byte(rand.Text())
	// 	r, w := io.Pipe()
	// 	rwc := &rwcadap{Reader: new(bytes.Buffer), Writer: w}
	// 	d.Dispatch(hash, rwc)
	// 	randPayload := make([]byte, 12)
	// 	expected, _ := model.NewSignal(
	// 		model.SignalTypeNeedInvite,
	// 		make([]byte, 32),
	// 		make([]byte, 32),
	// 		randPayload,
	// 	)
	// 	d.Send(expected)

	// 	wg := sync.WaitGroup{}
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()

	// 		buf := make([]byte, 1024)
	// 		n, err := r.Read(buf)
	// 		assert.NoError(t, err)

	// 		var s model.Signal
	// 		err = s.Unmarshal(buf[:n])
	// 		assert.NoError(t, err)
	// 		assert.Equal(t, expected, s)
	// 	}()
	// 	wg.Wait()
	// })
	t.Run("does not read", func(t *testing.T) {
		t.Parallel()
		d := New()
		hash := []byte(rand.Text())
		_, w := io.Pipe()
		isClosed := false
		var wg sync.WaitGroup
		rwc := &rwcadap{
			Reader: new(bytes.Buffer),
			Writer: w,
			close: func() error {
				isClosed = true
				return nil
			},
		}
		d.Dispatch(hash, rwc)
		randPayload := make([]byte, 12)
		expected, _ := model.NewSignal(
			model.SignalTypeNeedInvite,
			make([]byte, 32),
			make([]byte, 32),
			randPayload,
		)

		// Отправляем 257 сообщений
		for range 300 {
			d.Send(expected)
		}

		// Ждем завершения Close
		wg.Wait()
		assert.True(t, isClosed)
	})
}

func (r *rwcadap) Close() error {
	if r.close == nil {
		return nil
	}
	return r.close()
}
