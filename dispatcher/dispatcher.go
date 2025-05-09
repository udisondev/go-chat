package dispatcher

import (
	"context"
	"errors"
	"sync"
)

type SignalHandler func(*Distapcher, Signal) (Signal, error)

type Distapcher struct {
	mu       sync.RWMutex
	count    int
	nodes    map[string]Node
	handlers map[SignalType]SignalHandler
	maxCount int
}

func New(maxCount int, handlers map[SignalType]SignalHandler) Distapcher {
	return Distapcher{
		nodes:    make(map[string]Node, maxCount),
		handlers: handlers,
		maxCount: maxCount,
	}
}

type Node struct {
	outbox chan<- []byte
	close  func()
}

func (d *Distapcher) Dispatch(hash []byte, inbox <-chan []byte) (<-chan []byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.count++

	if d.count > d.maxCount {
		return nil, errors.New("busy")
	}

	outbox := make(chan []byte, 256)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer func() {
			d.mu.Lock()
			delete(d.nodes, string(hash))
			d.mu.Unlock()

			close(outbox)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case in, ok := <-inbox:
				if !ok {
					return
				}
				if len(in) < MinLen {
					return
				}
				s := Signal(in)
				h, ok := d.handlers[s.Type()]
				if !ok {
					continue
				}
				out, err := h(d, s)
				if err != nil {
					return
				}
				if out == nil {
					continue
				}

				d.send(out)
			}
		}

	}()

	d.nodes[string(hash)] = Node{
		outbox: outbox,
		close:  cancel,
	}

	return outbox, nil
}

func (d *Distapcher) send(o Signal) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	b := []byte(o)
	n, ok := d.nodes[o.RecipientString()]
	if ok {
		send(n, b)
		return
	}

	for _, n := range d.nodes {
		send(n, b)
	}
}

func send(n Node, b []byte) {
	select {
	case n.outbox <- b:
	default:
		n.close()
	}
}
