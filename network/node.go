package network

import (
	"context"
	"go-chat/conn"
	"go-chat/pkg/closer"
	"log"
	"net"
	"time"
)

func Attach(ctx context.Context, addr string, dspch conn.Dispatcher) error {
	d := net.Dialer{}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	err = conn.Upgrade(ctx, c, dspch)
	if err != nil {
		c.Close()
		return err
	}

	return nil
}

func Listen(
	addr string,
	upgradeTimeout time.Duration,
	dspch conn.Dispatcher,
) error {
	listenAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		return err
	}
	closer.Add(listener.Close)

	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				log.Printf("accept conn: %v", err)
				continue
			}
			go func() {
				ctx, close := context.WithTimeout(context.Background(), upgradeTimeout)
				defer close()

				err := conn.Upgrade(ctx, c, dspch)
				if err != nil {
					c.Close()
					log.Printf("interact with new conn: %v", err)
					return
				}
			}()
		}
	}()

	return nil
}
