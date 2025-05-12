package middleware

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"go-chat/config"
	"go-chat/pack"
	"io"
)

type Sumchecker struct {
	downstream io.ReadWriteCloser
	buf        []byte
}

func Checksum(rwc io.ReadWriteCloser) io.ReadWriteCloser {
	return &Sumchecker{
		downstream: rwc,
		buf:        make([]byte, config.MaxInputLen),
	}
}

func (s *Sumchecker) Read(b []byte) (int, error) {
	n, err := pack.ReadFrom(s.downstream, s.buf)
	if err != nil {
		return n, err
	}
	checksum, payload := s.buf[:sha256.Size], s.buf[sha256.Size:n]
	actual := sha256.Sum256(payload)
	if !bytes.Equal(checksum, actual[:]) {
		return 0, errors.New("invalid checksum")
	}
	return copy(b, payload), nil
}

func (s *Sumchecker) Write(b []byte) (int, error) {
	sum := sha256.Sum256(b)
	_, err := pack.WriteTo(s.downstream, append(sum[:], b...))
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (s *Sumchecker) Close() error {
	return s.downstream.Close()
}
