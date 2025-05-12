package main

import (
	"context"
	"flag"
	"go-chat/closer"
	"go-chat/dispatcher"
	"go-chat/handler"
	"go-chat/model"
	"go-chat/network"
	"time"
)

var (
	attachAddr = flag.String("attach", "", "Attach address")
	listenAddr = flag.String("attach", "", "Listen address")
)

func main() {
	flag.Parse()

	inbox := make(chan model.Signal)
	closer.Add(func() error { close(inbox); return nil })

	d := dispatcher.New()
	n := network.NewNode()
	handler.RunConnector(n, d)

	if attachAddr != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		p, err := n.Attach(ctx, *attachAddr)
		d.Dispatch(p.Hash(), p)

		if err != nil {
			panic(err)
		}
	}

	if listenAddr != nil {
		handler := func(p *network.Peer) {
			d.Dispatch(p.Hash(), p)
		}
		err := n.Listen(*listenAddr, time.Second*3, handler)
		if err != nil {
			panic(err)
		}
	}
}
