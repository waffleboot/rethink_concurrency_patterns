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

type Item struct{}

type Queue struct {
	s chan state
}

type waiter struct {
	n int
	c chan []Item
}

type state struct {
	items []Item
	wait  []waiter
}

func NewQueue() *Queue {
	s := make(chan state, 1)
	s <- state{}
	return &Queue{s}
}

func (q *Queue) Get(ctx context.Context) (Item, error) {
	var items []Item
	select {
	case <-ctx.Done():
		return Item{}, ctx.Err()
	case items = <-q.items:
	}
	item := items[0]
	items = items[1:]
	if len(items) == 0 {
		q.empty <- true
	} else {
		q.items <- items
	}
	return item, nil
}

func (q *Queue) GetMany(n int) []Item {

	s := <-q.s

	if len(s.wait) == 0 && len(s.items) >= n { // нет ожидающих и есть данные
		items := s.items[:n:n] // отрезаем данные и возвращаем состояние обратно в queue
		s.items = s.items[n:]
		q.s <- s
		return items
	}
	c := make(chan []Item)                // если данных нет или их недостаточно создаем канал
	s.wait = append(s.wait, waiter{n, c}) // включаем в список ожидающих и вешаемся на канале ожидания

	q.s <- s

	return <-c // не выходим, а вешаемся на канале ожидания
}

func (q *Queue) Put(item Item) {

	s := <-q.s

	s.items = append(s.items, item) // добавляем данные

	for len(s.wait) > 0 { // если есть ожидающие, иначе просто выходим
		w := s.wait[0]          // берем первого ожидающего
		if len(s.items) < w.n { // если данных не хватает, то просто выходим
			break
		}
		w.c <- s.items[:w.n:w.n] // а иначе отправляем в канал, но может ведь получится что никто и не читает канал
		s.items = s.items[w.n:]
		s.wait = s.wait[1:] // выключаем из режим ожидания
	}

	q.s <- s

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
