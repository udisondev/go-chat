package middleware

import (
	"crypto/ecdh"
	"go-chat/config"
	"go-chat/netcrypt"
	"io"
)

type Crypter struct {
	downstream io.ReadWriteCloser
	privkey    *ecdh.PrivateKey
	pubkey     *ecdh.PublicKey
	buf        []byte
}

func Crypt(privkey *ecdh.PrivateKey, pubkey *ecdh.PublicKey, rwc io.ReadWriteCloser) io.ReadWriteCloser {
	return &Crypter{
		downstream: rwc,
		privkey:    privkey,
		pubkey:     pubkey,
		buf:        make([]byte, config.MaxInputLen),
	}
}

func (c *Crypter) Read(b []byte) (int, error) {
	n, err := c.downstream.Read(c.buf)
	if err != nil {
		return 0, err
	}
	decrypted, err := netcrypt.Decrypt(c.buf[:n], c.privkey, c.pubkey)
	if err != nil {
		return 0, err
	}

	return copy(b, decrypted), nil
}

func (c *Crypter) Write(b []byte) (int, error) {
	encrypted, err := netcrypt.Encrypt(b, c.privkey, c.pubkey)
	if err != nil {
		return 0, err
	}
	_, err = c.downstream.Write(encrypted)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *Crypter) Close() error {
	return c.downstream.Close()
}
