package main

import (
	"sync"
)

type IdGenerator[T ~int64] struct {
	mu    sync.Mutex
	idMax T
}

func (gen *IdGenerator[T]) NewId() T {
	gen.mu.Lock()
	defer gen.mu.Unlock()

	gen.idMax += 1

	return gen.idMax
}

func (gen *IdGenerator[T]) MaxId() T {
	gen.mu.Lock()
	defer gen.mu.Unlock()

	return gen.idMax
}
