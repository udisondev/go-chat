package dispatcher

type Dispatcher struct {
	myID      string
	responder Responder
	inbox     chan []byte
	send      func(recepient string, out []byte)
}

func (d *Dispatcher) ServeInbox(inbox <-chan []byte) {
	go func() {
		for in := range inbox {
			d.inbox <- in
		}
	}()
}

func (d *Dispatcher) Run() {
	inbox := make(chan []byte)
	for in := range inbox {
		d.dispatch(in)
	}
}

func (d *Dispatcher) dispatch(in []byte) {
	s := Signal(in)
	if s.Type() == NeedInvite {
		out := d.responder.Handle(s)
		if out != nil {
			d.send(out.RecipientString(), out)
		}
		return
	}
	if s.RecipientString() != d.myID {
		d.send(s.RecipientString(), s)
		return
	}

	switch s.Type() {
	case ReadyToInvite:
	case WaitOffer:
	case WaitAnswer:
	case Answer:
	}
}
