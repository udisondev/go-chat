package main

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"flag"
	"go-chat/dispatcher"
	"go-chat/handler"
	"go-chat/network"
	"time"
)

var (
	attachAddr = flag.String("attach", "", "Attach address")
	listenAddr = flag.String("attach", "", "Listen address")
)

func main() {
	flag.Parse()

	privkey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	pubsign, privsign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	initiator := handler.Initiator{}

	handlers := map[dispatcher.SignalType]dispatcher.SignalHandler{
		dispatcher.RaiseYourHand: initiator.HandleRaiseYourHand,
	}

	dspch := dispatcher.New(handlers)

	n := network.NewNode(
		privkey,
		privsign,
		pubsign,
	)

	if attachAddr != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		err := n.Attach(ctx, *attachAddr, dspch.Dispatch)
		if err != nil {
			panic(err)
		}
	}

	if listenAddr != nil {
		dspch.SetEntrypoint(true)
		introducer := handler.Introducer{}
		err := n.Listen(*listenAddr, time.Second*2, dspch.Dispatch)
		if err != nil {
			panic(err)
		}
		dspch.AddHandler(dispatcher.Newbie, introducer.HandleNewbie)
	}
}
