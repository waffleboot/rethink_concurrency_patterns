package main

import "context"

type Item struct{}

type Queue struct {
	s chan state // хранит только один объект, отдается и возвращается горутинами
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

// раньше очередь содержала items канал под состояние с данными и empty семафор
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

	s := <-q.s // вытаскиваем состояние, другие горутины на этом зависнут т.к. канал будет пустой

	if len(s.wait) == 0 && len(s.items) >= n { // нет других ожидающих и есть данные
		items := s.items[:n:n] // отрезаем данные и возвращаем состояние обратно в queue
		s.items = s.items[n:]
		q.s <- s
		return items
	}
	c := make(chan []Item)                // если данных нет или их недостаточно создаем канал
	s.wait = append(s.wait, waiter{n, c}) // включаем в список ожидающих и вешаемся на канале ожидания

	q.s <- s // запихиваем состояние обратно

	return <-c // не выходим, а вешаемся на канале ожидания
}

func (q *Queue) Put(item Item) {

	s := <-q.s // завладеем состоянием

	s.items = append(s.items, item) // добавляем данные

	for len(s.wait) > 0 { // если есть ожидающие, иначе просто выходим
		w := s.wait[0]          // берем первого ожидающего
		if len(s.items) < w.n { // если данных все равно не хватает этому получателю, то просто выходим
			break
		}
		w.c <- s.items[:w.n:w.n] // а иначе отправляем в канал, см. GetMany return строчку
		s.items = s.items[w.n:]  // остаток оставляем на будущее
		s.wait = s.wait[1:]      // выключаем получателя из режим ожидания
	}

	q.s <- s // вернем состояние

}
