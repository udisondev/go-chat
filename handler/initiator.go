package handler

import "go-chat/dispatcher"

type Initiator struct {
}

func (i *Initiator) HandleRaiseYourHand(s dispatcher.Signal) (dispatcher.Signal, error) {
	return nil, nil
}
