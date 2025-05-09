package main

import (
	"context"
	"flag"
	"go-chat/config"
	"go-chat/dispatcher"
	"go-chat/network"
	"go-chat/upgrade"
	"time"
)

var (
	attachAddr = flag.String("attach", "", "Attach address")
	listenAddr = flag.String("attach", "", "Listen address")
)

func main() {
	flag.Parse()

	d := dispatcher.New(config.MazPeersCount, nil)

	u := upgrade.New(time.Second*5, &d)
	if attachAddr != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		err := network.Attach(ctx, *attachAddr, u.Upgrade)
		if err != nil {
			panic(err)
		}
	}

	if listenAddr != nil {
		err := network.Listen(*listenAddr, u.Upgrade)
		if err != nil {
			panic(err)
		}
	}
}
