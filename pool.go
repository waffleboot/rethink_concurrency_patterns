package main

import (
	"context"
	"net"
)

// Пул в качестве семафоров использует буферизованные каналы.
// Если в idle-канале есть объекты, то в acquire берутся из него.
// А sem канал заполняется токенами при каждом создании соединения и как только канал заполнится,
// то значит создавать лимит на количество соединений исчерпан, ждем либо hijack, либо пока в idle
// не появится свободное соединение.

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

func (p *Pool) Acquire(ctx context.Context) (net.Conn, error) {
	// либо берем из idle
	// либо создаем соединение пока можем писать в sem-канал, в семафор, пока его не исчерпаем
	// если захват из idle, то release вернет соединение обратно в канал
	// idle хранит только простаивающие соединения, рабочие отдаются во владение клиенту пула
	select {
	case conn := <-p.idle:
		return conn, nil
	case p.sem <- token{}:
		conn, err := dial()
		if err != nil {
			<-p.sem // не смогли создать соединение, значит только что созданный токен надо выкинуть
		}
		return conn, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *Pool) Release(c net.Conn) {
	p.idle <- c
}

// hijack это мы крадем соединение, т.е. оно перестает учитываться в лимитах, мы выкидываем один токен
// чтобы зависшая на acquire горутина могла создать новое соединение и вписать токен
func (p *Pool) Hijack(c net.Conn) {
	<-p.sem
}

func dial() (net.Conn, error) {
	return nil, nil
}
