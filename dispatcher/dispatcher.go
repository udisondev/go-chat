package dispatcher

import (
	"encoding/binary"
	"go-chat/config"
	"go-chat/network"
	"sync"
)

type peerWithLock struct {
	mu sync.Mutex
	p  *network.Peer
}

type Dispatcher struct {
	mu    sync.RWMutex
	peers map[string]*peerWithLock
}

func New() *Dispatcher {
	return &Dispatcher{
		peers: make(map[string]*peerWithLock, config.MazPeersCount),
	}
}

func (d *Dispatcher) InteractWith(p *network.Peer) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.peers[p.ID()] = &peerWithLock{p: p}

	go func() {
		defer func() {
			d.mu.Lock()
			defer d.mu.Unlock()
			delete(d.peers, p.ID())
			p.Close()
		}()

		buf := make([]byte, config.MaxInputLen)
		var mlen uint16
		for {
			binary.Read(p, binary.LittleEndian, &mlen)
			if len(buf) < int(mlen) {
				return
			}
			for read := 0; read < int(mlen); {
				n, err := p.Read(buf[read:])
				if err != nil {
					return
				}
				read += n
			}
			d.dispatch(buf[:mlen])
		}
	}()
}

func (d *Dispatcher) dispatch(b []byte) {

}

func (p *peerWithLock) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := binary.Write(p, binary.LittleEndian, uint16(len(b)))
	if err != nil {
		return 0, err
	}

	for written := 0; written < len(b); {
		n, err := p.Write(b[written:])
		if err != nil {
			return written, err
		}
		written += n
	}

	return len(b), nil
}
