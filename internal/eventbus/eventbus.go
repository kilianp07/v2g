package eventbus

import "sync"

// Event represents an arbitrary event passed on the bus.
type Event interface{}

// EventBus implements a simple publish/subscribe event bus.
type EventBus interface {
	Publish(Event)
	Subscribe() <-chan Event
	Unsubscribe(<-chan Event)
	Close()
}

// Bus is the default EventBus implementation using fan-out channels.
type Bus struct {
	mu     sync.RWMutex
	subs   []chan Event
	closed bool
}

// New creates a new Bus.
func New() *Bus { return &Bus{} }

// Publish sends the event to all subscribers. Delivery is non-blocking.
func (b *Bus) Publish(e Event) {
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

// Subscribe registers a new subscriber and returns its channel.
func (b *Bus) Subscribe() <-chan Event {
	ch := make(chan Event, 8)
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
func (b *Bus) Unsubscribe(sub <-chan Event) {
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

// Close closes all subscriber channels and clears the list.
func (b *Bus) Close() {
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
