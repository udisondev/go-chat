package middleware

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/binary"
	"go-chat/pkg/crypto"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Crypto(t *testing.T) {
	pprivKey, err := ecdh.P256().GenerateKey(rand.Reader)
	assert.NoError(t, err)
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	assert.NoError(t, err)
	t.Run("success decrypt text", func(t *testing.T) {
		dr, dw := io.Pipe()
		rw := RWWrapper{
			Reader: dr,
		}
		enigma := Crypto(privKey, pprivKey.PublicKey(), &rw)

		text := rand.Text()
		encrypted, err := crypto.Encrypt([]byte(text), pprivKey, privKey.PublicKey())
		assert.NoError(t, err)
		go func() {
			err := binary.Write(dw, binary.LittleEndian, uint16(len(encrypted)))
			assert.NoError(t, err)
			_, err = dw.Write(encrypted)
			assert.NoError(t, err)
		}()

		var mlen uint16
		err = binary.Read(enigma, binary.LittleEndian, &mlen)
		assert.NoError(t, err)
		buf := make([]byte, mlen)
		_, err = enigma.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, []byte(text), buf)
	})

}
