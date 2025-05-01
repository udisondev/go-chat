package middleware

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_readDownstream(t *testing.T) {
	t.Run("Success read", func(t *testing.T) {
		text := rand.Text()
		buf := make([]byte, 1024)

		dr, dw := io.Pipe()
		ur, uw := io.Pipe()

		go func() {
			mlen := len(text)
			err := binary.Write(dw, binary.LittleEndian, uint16(mlen))
			assert.NoError(t, err)
			_, err = dw.Write([]byte(text))
			assert.NoError(t, err)
		}()

		go func() {
			var mlen uint16
			err := binary.Read(ur, binary.LittleEndian, &mlen)
			assert.NoError(t, err)
			assert.Equal(t, mlen, uint16(len(text)))
			buf := make([]byte, mlen)
			for read := 0; read < int(mlen); {
				n, err := ur.Read(buf[read:])
				if err != nil {
					t.Error(err)
				}
				read += n
			}
			assert.Equal(t, []byte(text), buf)
		}()

		err := readDownstream(buf, 1, dr, uw, func(b []byte) ([]byte, error) {
			assert.Equal(t, []byte(text), b)
			return b, nil
		})
		assert.NoError(t, err)
	})

	t.Run("handler error", func(t *testing.T) {
		text := rand.Text()
		buf := make([]byte, 1024)

		dr, dw := io.Pipe()
		_, uw := io.Pipe()

		go func() {
			mlen := len(text)
			err := binary.Write(dw, binary.LittleEndian, uint16(mlen))
			assert.NoError(t, err)
			_, err = dw.Write([]byte(text))
			assert.NoError(t, err)
		}()
		err := readDownstream(buf, 1, dr, uw, func(b []byte) ([]byte, error) {
			assert.Equal(t, []byte(text), b)
			return nil, errors.New("handler error")
		})
		assert.Error(t, err)
	})

	t.Run("min len error", func(t *testing.T) {
		text := rand.Text()
		buf := make([]byte, 1024)

		dr, dw := io.Pipe()
		_, uw := io.Pipe()

		go func() {
			mlen := len(text)
			err := binary.Write(dw, binary.LittleEndian, uint16(mlen))
			assert.NoError(t, err)
			_, err = dw.Write([]byte(text))
			assert.NoError(t, err)
		}()
		err := readDownstream(buf, 2000, dr, uw, nil)
		assert.ErrorContains(t, err, "too short message")
	})

	t.Run("over len error", func(t *testing.T) {
		text := rand.Text()
		buf := make([]byte, 20)

		dr, dw := io.Pipe()
		_, uw := io.Pipe()

		go func() {
			mlen := len(text)
			err := binary.Write(dw, binary.LittleEndian, uint16(mlen))
			assert.NoError(t, err)
			_, err = dw.Write([]byte(text))
			assert.NoError(t, err)
		}()
		err := readDownstream(buf, 1, dr, uw, nil)
		assert.ErrorContains(t, err, "too big message")
	})
}
