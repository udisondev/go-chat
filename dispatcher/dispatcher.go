package dispatcher

import (
	"crypto/rand"
	"encoding/binary"
	"go-chat/config"
	"go-chat/network"
	"sync"
	"unsafe"
)

type peerWithLock struct {
	mu sync.Mutex
	p  *network.Peer
}

type Dispatcher struct {
	mu       sync.RWMutex
	peers    map[string]*peerWithLock
	cachePut func(string)
}

func New(cachePut func(string)) *Dispatcher {
	return &Dispatcher{
		peers:    make(map[string]*peerWithLock, config.MazPeersCount),
		cachePut: cachePut,
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

func (d *Dispatcher) write(recipient string, b []byte) {
	payload := make([]byte, 12+len(b))
	copy(payload[12:], b)

	rand.Read(payload[:12])
	d.cachePut(unsafe.String(&payload[0], 12))

	d.mu.Lock()
	defer d.mu.Unlock()

	p, ok := d.peers[recipient]
	if ok {
		p.mu.Lock()
		defer p.mu.Unlock()

		_, err := p.Write(payload)
		if err != nil {
			p.p.Close()
			delete(d.peers, recipient)
		}
		return
	}

	for id, p := range d.peers {
		go func() {
			p.mu.Lock()
			defer p.mu.Unlock()

			_, err := p.Write(payload)
			if err != nil {
				p.p.Close()
				delete(d.peers, id)
			}
		}()
	}
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
