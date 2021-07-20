package main

import (
	"context"
	"log"
	"net"
)

type token struct{}

type Pool struct {
	sem  chan token
	idle chan net.Conn
}

func NewPool(limit int) *Pool {
	sem := make(chan token, limit)
	idle := make(chan net.Conn, limit)
	return &Pool{sem, idle}
}

func (p *Pool) Release(c net.Conn) {
	p.idle <- c
}

// hijack это типа мы крадем соединение, т.е. оно перестает учитываться в лимитах
func (p *Pool) Hijack(c net.Conn) {
	<-p.sem
}

func (p *Pool) Acquire(ctx context.Context) (net.Conn, error) {
	select {
	case conn := <-p.idle:
		return conn, nil
	case p.sem <- token{}:
		conn, err := dial()
		if err != nil {
			<-p.sem
		}
		return conn, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func dial() (net.Conn, error) {
	return nil, nil
}

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
