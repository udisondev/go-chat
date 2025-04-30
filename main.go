package main

import (
	"context"
	"flag"
	"go-chat/cache"
	"go-chat/dispatcher"
	"go-chat/middleware"
	"go-chat/network"
	"time"
)

var (
	attachAddr = flag.String("attach", "", "Attach address")
	listenAddr = flag.String("attach", "", "Listen address")
)

func main() {
	flag.Parse()

	csh := cache.New()
	filter := middleware.Filter(csh.Put, csh.PutIfAbsent)

	dsptch := dispatcher.New()

	if attachAddr != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		p, err := network.Attach(ctx, *attachAddr, filter)
		if err != nil {
			panic(err)
		}
		dsptch.InteractWith(p)
	}

	if listenAddr != nil {
		network.Listen(*listenAddr, time.Second*2, dsptch.InteractWith, filter)
	}
}
