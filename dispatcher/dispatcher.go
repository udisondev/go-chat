package dispatcher

import (
	"context"
	"go-chat/cache"
	"go-chat/config"
	"sync"
	"sync/atomic"
	"time"
)

type peer struct {
	disconnect func()
	send       func([]byte)
}

type peerState uint8

const (
	NewBie peerState = iota
	Trusted
)

type Client struct {
	hash     []byte
	cache    *cache.Cache
	handlers map[SignalType]func(Signal) (Signal, error)
}

type Server struct {
	hash      []byte
	peersMu   sync.Mutex
	peers     map[string]*peer
	newbiesMu sync.Mutex
	newbies   map[string]*peer
	handlers  map[SignalType]func(Signal) (Signal, error)
	cache     *cache.Cache
}

func NewClient(hash []byte) *Client {
	return &Client{
		hash:     hash,
		cache:    cache.New(config.CacheBucketsCount, config.CacheBucketSize),
		handlers: make(map[SignalType]func(Signal) (Signal, error)),
	}
}

func NewServer(hash []byte) *Server {
	return &Server{
		hash:     hash,
		cache:    cache.New(config.CacheBucketsCount, config.CacheBucketSize),
		peers:    make(map[string]*peer),
		newbies:  make(map[string]*peer),
		handlers: make(map[SignalType]func(Signal) (Signal, error)),
	}
}

func (d *Client) Dispatch(_ []byte, input <-chan []byte) <-chan []byte {
	output := make(chan []byte, 256)
	go func() {
		defer close(output)
		for in := range input {
			s := Signal(in)
			if !d.cache.PutIfAbsent(s.Nonce()) {
				continue
			}
		}
	}()

	return output
}

func (d *Server) Dispatch(hash []byte, input <-chan []byte) <-chan []byte {
	strhash := string(hash)
	d.newbiesMu.Lock()
	d.newbiesMu.Unlock()
	output := make(chan []byte, 256)
	ctx, closeCtx := context.WithCancel(context.Background())
	disconnected := atomic.Bool{}
	mu := sync.Mutex{}
	disconnect := sync.OnceFunc(func() {
		closeCtx()

		d.peersMu.Lock()
		defer d.peersMu.Unlock()

		mu.Lock()
		defer mu.Unlock()
		disconnected.Swap(false)

		delete(d.peers, strhash)
		close(output)
	})

	p := peer{
		disconnect: disconnect,
		send: func(b []byte) {
			mu.Lock()
			defer mu.Unlock()
			if disconnected.Load() {
				return
			}
			select {
			case output <- b:
			default:
				go disconnect()
			}
		},
	}

	d.newbies[strhash] = &p
	go func() {
		<-time.After(time.Second * 10)
		d.newbiesMu.Lock()
		defer d.newbiesMu.Unlock()

		p, ok := d.newbies[strhash]
		if !ok {
			return
		}
		p.disconnect()
		delete(d.newbies, strhash)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case in, ok := <-input:
				if !ok {
					return
				}
				s := Signal(in)
				if !d.cache.PutIfAbsent(s.Nonce()) {
					continue
				}
			}
		}
	}()

	return output
}
