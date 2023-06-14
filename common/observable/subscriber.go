//go:build foss

package observable

import (
	"sync"
)

type Subscription[T any] <-chan T

type Subscriber[T any] struct {
	buffer chan T
	once   sync.Once
}

func (s *Subscriber[T]) Emit(item T) {
	select {
	case s.buffer <- item:
	default:
	}
}

func (s *Subscriber[T]) Out() Subscription[T] {
	return s.buffer
}

func (s *Subscriber[T]) Close() {
	s.once.Do(func() {
		close(s.buffer)
	})
}

func newSubscriber[T any]() *Subscriber[T] {
	return &Subscriber[T]{
		buffer: make(chan T, 200),
	}
}
