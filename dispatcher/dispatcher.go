package dispatcher

import (
	"fmt"
	"go-chat/config"
	"sync"
	"sync/atomic"
)

type peer struct {
	mu         sync.Mutex
	disconnect func()
	send       func([]byte)
}

type Dispatcher struct {
	mu       sync.RWMutex
	peers    map[string]*peer
	cachePut func(string)
}

func New(cachePut func(string)) *Dispatcher {
	return &Dispatcher{
		peers:    make(map[string]*peer, config.MazPeersCount),
		cachePut: cachePut,
	}
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
		send: func(b []byte) {
			mu.Lock()
			defer mu.Unlock()

			if disconnected.Load() {
				return
			}
			select {
			case output <- b:
			default:
				disconnect()
			}
		},
	}

	go func() {
		defer close(output)

		for in := range input {
			fmt.Println(string(in))
		}
	}()

	return output
}
