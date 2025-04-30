package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Checksum(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dr, dw := io.Pipe()
		rw := RWWrapper{
			Reader: dr,
		}
		checksumer := Checksum(rw)

		text := rand.Text()
		b_text := []byte(text)
		sum := sha256.Sum256(b_text)
		go func() {
			payload := append(sum[:], b_text...)
			err := binary.Write(dw, binary.LittleEndian, uint16(len(payload)))
			assert.NoError(t, err)
			_, err = dw.Write(payload)
			assert.NoError(t, err)
		}()

		var mlen uint16
		err := binary.Read(checksumer, binary.LittleEndian, &mlen)
		assert.NoError(t, err)
		buf := make([]byte, mlen)
		_, err = checksumer.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, b_text, buf)
	})

	t.Run("invalid", func(t *testing.T) {
		dr, dw := io.Pipe()
		rw := RWWrapper{
			Reader: dr,
		}

		checksumer := Checksum(rw)

		text := rand.Text()
		b_text := []byte(text)
		sum := sha256.Sum256([]byte(rand.Text()))
		go func() {
			payload := append(sum[:], b_text...)
			err := binary.Write(dw, binary.LittleEndian, uint16(len(payload)))
			assert.NoError(t, err)
			_, err = dw.Write(payload)
			assert.NoError(t, err)
		}()

		var mlen uint16
		err := binary.Read(checksumer, binary.LittleEndian, &mlen)
		assert.Error(t, err)
	})

	t.Run("write checksum", func(t *testing.T) {
		bb := new(bytes.Buffer)
		checksumer := Checksum(bb)

		text := rand.Text()
		sum := sha256.Sum256([]byte(text))
		expected := append(sum[:], []byte(text)...)
		_, err := checksumer.Write([]byte(text))
		assert.NoError(t, err)
		var mlen uint16
		err = binary.Read(bb, binary.LittleEndian, &mlen)
		buf := make([]byte, mlen)
		_, err = bb.Read(buf)
		assert.Equal(t, buf, expected)
	})
}
