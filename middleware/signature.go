package middleware

import (
	"crypto/ed25519"
	"errors"
	"go-chat/config"
	"io"
)

type SignChecker struct {
	privsign   ed25519.PrivateKey
	pubsign    ed25519.PublicKey
	downstream io.ReadWriteCloser
	buf        []byte
}

func SignCheck(privsign ed25519.PrivateKey, pubsign ed25519.PublicKey, rwc io.ReadWriteCloser) io.ReadWriteCloser {
	return &SignChecker{
		privsign:   privsign,
		pubsign:    pubsign,
		downstream: rwc,
		buf:        make([]byte, config.MaxInputLen),
	}
}
func (s *SignChecker) Read(b []byte) (int, error) {
	n, err := s.downstream.Read(s.buf)
	if err != nil {
		return 0, err
	}
	signature, payload := s.buf[:ed25519.SignatureSize], s.buf[ed25519.SignatureSize:n]
	if !ed25519.Verify(s.pubsign, payload, signature) {
		return 0, errors.New("invalid sign")
	}
	return copy(b, payload), nil
}

func (s *SignChecker) Write(b []byte) (int, error) {
	_, err := s.downstream.Write(append(ed25519.Sign(s.privsign, b), b...))
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (s *SignChecker) Close() error {
	return s.downstream.Close()
}
