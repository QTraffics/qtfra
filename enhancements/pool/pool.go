package pool

import (
	"sync"
)

type Pool[T any] struct {
	pool *sync.Pool
}

func New[T any](constructor func() T) *Pool[T] {
	return &Pool[T]{
		pool: &sync.Pool{New: func() any {
			return constructor()
		}},
	}
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool[T]) Put(v T) {
	p.pool.Put(v)
}
