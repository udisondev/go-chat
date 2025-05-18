package dispatcher

import (
	"context"
	"go-chat/config"
	"go-chat/model"
	"io"
	"sync"
)

type Dispatcher struct {
	mu       sync.Mutex
	peers    map[string]*Node
	typemu   sync.Mutex
	typesubs map[model.SignalType][]chan model.Signal
	keymu    sync.Mutex
	keysubs  map[string]chan model.Signal
}

type Node struct {
	close  func()
	outbox chan<- []byte
}

func New() *Dispatcher {
	return &Dispatcher{
		peers:    map[string]*Node{},
		typesubs: map[model.SignalType][]chan model.Signal{},
	}
}

func (d *Dispatcher) SubscribeType(st model.SignalType) <-chan model.Signal {
	d.typemu.Lock()
	defer d.typemu.Unlock()

	subs, ok := d.typesubs[st]
	if !ok {
		subs = make([]chan model.Signal, 0, 1)
	}
	x := make(chan model.Signal, 100)

	subs = append(subs, x)
	d.typesubs[st] = subs

	return x
}

func (d *Dispatcher) SubscribeKey(key string) <-chan model.Signal {
	d.keymu.Lock()
	defer d.keymu.Unlock()

	ch := make(chan model.Signal, 100)
	d.keysubs[key] = ch
	return ch
}

func (d *Dispatcher) UnsbribeKey(key string) {
	d.keymu.Lock()
	defer d.keymu.Unlock()

	s, ok := d.keysubs[key]
	if !ok {
		return
	}

	close(s)
	delete(d.keysubs, key)
}

func (d *Dispatcher) Dispatch(hash []byte, rwc io.ReadWriteCloser) {
	d.mu.Lock()
	defer d.mu.Unlock()

	outbox := make(chan []byte, 256)

	ctx, cancel := context.WithCancel(context.Background())
	d.peers[string(hash)] = &Node{
		close:  cancel,
		outbox: outbox,
	}

	go func() {
		defer func() {
			d.mu.Lock()
			defer d.mu.Unlock()
			delete(d.peers, string(hash))
			rwc.Close()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case out, ok := <-outbox:
				if !ok {
					return
				}
				_, err := rwc.Write(out)
				if err != nil {
					return
				}
			}
		}
	}()

	go func() {
		defer cancel()

		buf := make([]byte, config.MaxInputLen)
		for {
			n, err := rwc.Read(buf)
			if err != nil {
				return
			}
			tmp := make([]byte, n)
			copy(tmp, buf[:n])
			s, err := model.FormatSignal(tmp)
			if err != nil {
				return
			}

			for _, typesub := range d.typesubs[s.Type()] {
				typesub <- s
			}

			keysub, ok := d.keysubs[s.KeyString()]
			if !ok {
				continue
			}

			keysub <- s
		}
	}()
}

func (d *Dispatcher) Disconnect(hash []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n, ok := d.peers[string(hash)]
	if !ok {
		return
	}
	n.close()
}

func (d *Dispatcher) Send(s model.Signal) {
	d.mu.Lock()
	defer d.mu.Unlock()

	b := []byte(s)
	n, ok := d.peers[s.RecipientString()]
	if ok {
		send(n, b)
		return
	}

	for _, n := range d.peers {
		send(n, b)
	}
}

func send(n *Node, b []byte) {
	select {
	case n.outbox <- b:
	default:
		n.close()
	}
}
