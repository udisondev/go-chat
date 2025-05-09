package network

import (
	"context"
	"go-chat/pkg/closer"
	"io"
	"log"
	"net"
)

type Upgrader func(io.ReadWriteCloser) error

func Attach(ctx context.Context, addr string, upgr Upgrader) error {
	d := net.Dialer{}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	err = upgr(c)
	if err != nil {
		c.Close()
		return err
	}

	return nil
}

func Listen(addr string, upgr Upgrader) error {
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
				err := upgr(c)
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
