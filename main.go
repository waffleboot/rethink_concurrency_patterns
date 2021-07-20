package main

import (
	"log"
	"net"
	"sync"
)

type Pool struct {
	mu              sync.Mutex
	cond            sync.Cond
	numConns, limit int
	idle            []net.Conn
}

func NewPool(limit int) *Pool {
	p := &Pool{limit: limit}
	p.cond.L = &p.mu
	return p
}

func (p *Pool) Release(c net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.idle = append(p.idle, c)
	p.cond.Signal()
}

// hijack это типа мы крадем соединение, т.е. оно перестает учитываться в лимитах
func (p *Pool) Hijack(c net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.numConns--
	p.cond.Signal()
}

func (p *Pool) Acquire() (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.idle) == 0 && p.numConns >= p.limit {
		p.cond.Wait()
	}
	if len(p.idle) > 0 {
		c := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
		return c, nil
	}
	c, err := dial()
	if err == nil {
		p.numConns++
	}
	return c, err
}

func dial() (net.Conn, error) {
	return nil, nil
}

/*
т.е. мы сразу отдаем
*/

func main() {
	p := NewPool(3)
	c, err := p.Acquire()
	if err != nil {
		log.Fatal(err)
	}
	p.Release(c)
}
