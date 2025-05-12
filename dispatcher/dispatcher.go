package dispatcher

import (
	"context"
	"go-chat/config"
	"go-chat/model"
	"io"
	"sync"
)

type Dispatcher struct {
	mu     sync.Mutex
	peers  map[string]*Node
	topics map[model.SignalType][]chan model.Signal
}

type Node struct {
	close  func()
	outbox chan<- []byte
}

func New() *Dispatcher {
	return &Dispatcher{topics: map[model.SignalType][]chan model.Signal{}}
}

func (d *Dispatcher) Subscribe(st model.SignalType) <-chan model.Signal {
	subs, ok := d.topics[st]
	if !ok {
		subs = make([]chan model.Signal, 0, 1)
	}
	x := make(chan model.Signal, 100)

	subs = append(subs, x)
	d.topics[st] = subs

	return x
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

			for _, t := range d.topics[s.Type] {
				t <- s
			}
		}
	}()
}

func (d *Dispatcher) Disconnect(hash string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n, ok := d.peers[hash]
	if !ok {
		return
	}
	n.close()
}

func (d *Dispatcher) Send(s model.Signal) {
	b, err := s.Marshal()
	if err != nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()
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
