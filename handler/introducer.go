package handler

import "go-chat/dispatcher"

type Introducer struct {
	id []byte
}

func (i *Introducer) HandleNewbie(s dispatcher.Signal) (dispatcher.Signal, error) {
	return nil, nil
}
