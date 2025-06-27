package eventbus

import "sync"

// TypedBus is a type-safe publish/subscribe bus for events of type T.
type TypedBus[T any] struct {
	mu     sync.RWMutex
	subs   []chan T
	closed bool
}

// NewTyped creates a new TypedBus.
func NewTyped[T any]() *TypedBus[T] { return &TypedBus[T]{} }

// Publish sends the event to all subscribers. Delivery is non-blocking.
func (b *TypedBus[T]) Publish(e T) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

// Subscribe registers a subscriber and returns its channel.
func (b *TypedBus[T]) Subscribe() <-chan T {
	ch := make(chan T, 8)
	b.mu.Lock()
	if b.closed {
		close(ch)
	} else {
		b.subs = append(b.subs, ch)
	}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the subscriber and closes its channel.
func (b *TypedBus[T]) Unsubscribe(sub <-chan T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, ch := range b.subs {
		if ch == sub {
			b.subs = append(b.subs[:i], b.subs[i+1:]...)
			if !b.closed {
				close(ch)
			}
			return
		}
	}
}

// Close closes the bus and all subscriber channels.
func (b *TypedBus[T]) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	for _, ch := range b.subs {
		close(ch)
	}
	b.subs = nil
	b.mu.Unlock()
}
