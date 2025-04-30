package network

import (
	"encoding/binary"
	"errors"
	"go-chat/middleware"
	"io"
	"net"
	"sync"
)

type Peer struct {
	mu   sync.Mutex
	conn net.Conn
	rw   io.ReadWriter
}

func NewPeer(conn net.Conn, mws ...middleware.Middleware) *Peer {
	var rw io.ReadWriter = conn
	for _, mw := range mws {
		rw = mw(rw)
	}
	return &Peer{
		conn: conn,
		rw:   rw,
	}
}

func (p *Peer) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var mlen uint16
	err := binary.Read(p.rw, binary.LittleEndian, &mlen)
	if err != nil {
		return 0, err
	}
	if len(b) < int(mlen) {
		return 0, errors.New("too big message")
	}
	read := 0
	for read < int(mlen) {
		n, err := p.rw.Read(b[read:])
		if err != nil {
			return read, err
		}
		read += n
	}
	return read, nil
}

func (p *Peer) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := binary.Write(p.rw, binary.LittleEndian, uint16(len(b)))
	if err != nil {
		return 0, err
	}
	written := 0
	for written < len(b) {
		n, err := p.rw.Write(b[written:])
		if err != nil {
			return written, err
		}
		written += n
	}
	return written, nil
}

func (p *Peer) Close() error {
	return p.conn.Close()
}
