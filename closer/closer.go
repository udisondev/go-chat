package closer

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	mu  sync.Mutex
	fns []func() error
)

func init() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-sig
		for _, fn := range fns {
			err := fn()
			if err != nil {
				log.Printf("closing err: %v", err)
			}
		}
	}()
}

func Add(fn func() error) {
	mu.Lock()
	defer mu.Unlock()

	fns = append(fns, fn)
}
