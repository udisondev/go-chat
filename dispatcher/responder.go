package dispatcher

type Responder struct {
}

func (r *Responder) Handle(s Signal) Signal {
	return nil
}
