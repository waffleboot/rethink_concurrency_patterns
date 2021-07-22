package main

import (
	"context"
	"log"
)

/*
т.е. мы сразу отдаем
*/

func main() {
	p := NewPool(3)
	c, err := p.Acquire(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	p.Release(c)
}

type Idler struct {
	next chan chan struct{}
}

func (i *Idler) AwaitIdle(ctx context.Context) error {
	idle := <-i.next
	i.next <- idle
	if idle != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-idle:
		}
	}
	return nil
}

func (i *Idler) SetBusy(b bool) {
	idle := <-i.next
	if b && (idle == nil) {
		idle = make(chan struct{})
	} else if !b && (idle != nil) {
		close(idle)
		idle = nil
	}
	i.next <- idle
}

func NewIdler() *Idler {
	next := make(chan chan struct{}, 1)
	next <- nil
	return &Idler{next}
}
