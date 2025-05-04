package dispatcher

import (
	"errors"
	"go-chat/cache"
	"go-chat/config"
	"sync"
	"sync/atomic"
)

type peer struct {
	mu         sync.Mutex
	disconnect func()
	send       func([]byte) error
}

type Dispatcher struct {
	mu       sync.RWMutex
	hash     []byte
	peers    map[string]*peer
	cache    *cache.Cache
	handlers map[SignalType]SignalHandler
}

type SignalHandler func(Signal) (Signal, error)

func New(handlers map[SignalType]SignalHandler) *Dispatcher {
	return &Dispatcher{
		peers:    make(map[string]*peer, config.MazPeersCount),
		cache:    cache.New(config.CacheBucketsCount, config.CacheBucketSize),
		handlers: handlers,
	}
}

func (d *Dispatcher) AddHandler(t SignalType, h SignalHandler) {
	d.handlers[t] = h
}

func (d *Dispatcher) Dispatch(hash string, input <-chan []byte) <-chan []byte {
	output := make(chan []byte)
	mu := new(sync.Mutex)
	disconnected := atomic.Bool{}
	disconnect := sync.OnceFunc(func() {
		mu.Lock()
		defer mu.Unlock()

		disconnected.Swap(true)
		close(output)
	})

	d.mu.Lock()
	d.peers[hash] = &peer{
		disconnect: disconnect,
		send: func(b []byte) error {
			mu.Lock()
			defer mu.Unlock()

			if disconnected.Load() {
				return nil
			}

			select {
			case output <- b:
				return nil
			default:
				disconnect()
				return errors.New("don't read!")
			}
		},
	}

	go func() {
		defer close(output)

		for in := range input {
			s := Signal(in)
			if !d.cache.PutIfAbsent(s.Nonce()) {
				continue
			}

			h, ok := d.handlers[s.Type()]
			if !ok {
				continue
			}
			out, err := h(s)
			if err != nil {
				return
			}
			if out.Payload == nil {
				continue
			}
			d.send(out)
		}
	}()

	return output
}

func (d *Dispatcher) send(s Signal) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache.Put(s.Nonce())
	p, ok := d.peers[s.RecipientString()]
	if ok {
		p.mu.Lock()
		defer p.mu.Unlock()

		p.send([]byte(s))
		return
	}

	for h, p := range d.peers {
		err := p.send([]byte(s))
		if err == nil {
			continue
		}
		delete(d.peers, h)
	}
}
