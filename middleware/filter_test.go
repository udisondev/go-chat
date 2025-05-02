package middleware

import (
	"crypto/rand"
	"encoding/binary"
	"go-chat/config"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Filter(t *testing.T) {
	type adapter struct {
		io.Reader
		io.Writer
	}
	t.Run("success nonce not exists", func(t *testing.T) {
		dr, dw := io.Pipe()
		a := adapter{
			Reader: dr,
		}
		nonce := make([]byte, config.NonceLen)
		text := rand.Text()
		rand.Read(nonce)
		payload := append(nonce, []byte(text)...)

		exists := func(s string) bool {
			assert.Equal(t, s, string(nonce))
			return false
		}
		filter := Filter(exists)(a)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := binary.Write(dw, binary.LittleEndian, uint16(len(payload)))
			assert.NoError(t, err)
			_, err = dw.Write(payload)
			assert.NoError(t, err)
		}()

		var mlen uint16
		err := binary.Read(filter, binary.LittleEndian, &mlen)
		assert.NoError(t, err)
		buf := make([]byte, mlen)
		_, err = filter.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, []byte(text), buf)
		wg.Wait()
	})
}
