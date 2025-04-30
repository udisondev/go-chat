package network

import (
	"go-chat/middleware"
	"io"
	"net"
)

type Peer struct {
	id   string
	conn net.Conn
	io.ReadWriter
}

func NewPeer(ID string, conn net.Conn, mws ...middleware.Middleware) *Peer {
	var rw io.ReadWriter = conn
	for _, mw := range mws {
		rw = mw(rw)
	}
	return &Peer{
		id:         ID,
		conn:       conn,
		ReadWriter: rw,
	}
}

func (p *Peer) ID() string {
	return p.id
}

func (p *Peer) Close() error {
	return p.conn.Close()
}
