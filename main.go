package main

import (
	"context"
	"flag"
	"go-chat/cache"
	"go-chat/config"
	"go-chat/dispatcher"
	"go-chat/network"
	"time"
)

var (
	attachAddr = flag.String("attach", "", "Attach address")
	listenAddr = flag.String("attach", "", "Listen address")
)

func main() {
	flag.Parse()

	csh := cache.New(config.CacheBucketsCount, config.CacheBucketSize)
	dsptch := dispatcher.New(csh.Put)

	if attachAddr != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		err := network.Attach(ctx, *attachAddr, dsptch.Dispatch)
		if err != nil {
			panic(err)
		}
	}

	if listenAddr != nil {
		err := network.Listen(*listenAddr, time.Second*2, dsptch.Dispatch)
		if err != nil {
			panic(err)
		}
	}
}
