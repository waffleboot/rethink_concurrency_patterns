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
