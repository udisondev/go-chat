package middleware

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Signature(t *testing.T) {
	ppubSign, pprivSign, _ := ed25519.GenerateKey(rand.Reader)
	pubSign, privSign, _ := ed25519.GenerateKey(rand.Reader)
	t.Run("success read signed pack", func(t *testing.T) {
		dr, dw := io.Pipe()
		rw := RWWrapper{
			Reader: dr,
		}
		signer := Signature(privSign, ppubSign, &rw)

		text := rand.Text()
		signature := ed25519.Sign(pprivSign, []byte(text))
		payload := append(signature, []byte(text)...)
		go func() {
			err := binary.Write(dw, binary.LittleEndian, uint16(len(payload)))
			assert.NoError(t, err)
			_, err = dw.Write(payload)
			assert.NoError(t, err)
		}()

		var mlen uint16
		err := binary.Read(signer, binary.LittleEndian, &mlen)
		assert.NoError(t, err)
		buf := make([]byte, mlen)
		_, err = signer.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, []byte(text), buf)
	})

	t.Run("sign", func(t *testing.T) {
		text := rand.Text()
		bb := new(bytes.Buffer)
		signer := Signature(privSign, ppubSign, bb)

		_, err := signer.Write([]byte(text))
		assert.NoError(t, err)
		var mlen uint16
		err = binary.Read(bb, binary.LittleEndian, &mlen)
		input := make([]byte, mlen)
		_, err = bb.Read(input)
		signature, payload := input[:ed25519.SignatureSize], input[ed25519.SignatureSize:]
		if !ed25519.Verify(pubSign, payload, signature) {
			t.Error("Failed sing")
		}
	})

}
