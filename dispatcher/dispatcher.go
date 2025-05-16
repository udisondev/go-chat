package dispatcher

import (
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
	return &Dispatcher{
		peers:  map[string]*Node{},
		topics: map[model.SignalType][]chan model.Signal{},
	}
}

func (d *Dispatcher) Subscribe(st model.SignalType) <-chan model.Signal {
	subs, ok := d.topics[st]
	if !ok {
		subs = make([]chan model.Signal, 0, 1)
	}
	ch := make(chan model.Signal, 100)

	subs = append(subs, ch)
	d.topics[st] = subs

	return ch
}

func (d *Dispatcher) Dispatch(hash []byte, rwc io.ReadWriteCloser) {
	d.mu.Lock()
	defer d.mu.Unlock()

	outbox := make(chan []byte, 256)
	stop := sync.OnceFunc(func() {
		rwc.Close()
	})

	d.peers[string(hash)] = &Node{
		close:  stop,
		outbox: outbox,
	}

	go func() {
		defer func() {
			d.mu.Lock()
			defer d.mu.Unlock()
			delete(d.peers, string(hash))
		}()

		for out := range outbox {
			_, err := rwc.Write(out)
			if err != nil {
				return
			}
		}
	}()

	go func() {
		defer stop()

		buf := make([]byte, config.MaxInputLen)
		for {
			n, err := rwc.Read(buf)
			if err != nil {
				return
			}
			tmp := make([]byte, n)
			copy(tmp, buf[:n])
			var s model.Signal
			err = s.Unmarshal(tmp)
			if err != nil {
				return
			}

			for _, t := range d.topics[s.Type] {
				t <- s
			}
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
