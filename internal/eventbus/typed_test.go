package eventbus

import "testing"

func TestTypedBusPublishSubscribe(t *testing.T) {
	bus := NewTyped[string]()
	ch := bus.Subscribe()
	bus.Publish("hello")
	v := <-ch
	if v != "hello" {
		t.Fatalf("expected hello got %v", v)
	}
	bus.Unsubscribe(ch)
}

func TestTypedBusClose(t *testing.T) {
	bus := NewTyped[int]()
	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()
	bus.Close()
	if _, ok := <-ch1; ok {
		t.Fatalf("expected ch1 closed")
	}
	if _, ok := <-ch2; ok {
		t.Fatalf("expected ch2 closed")
	}
}

func TestTypedBusUnsubscribeAfterClose(t *testing.T) {
	bus := NewTyped[float64]()
	ch := bus.Subscribe()
	bus.Close()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on Unsubscribe after Close: %v", r)
		}
	}()
	bus.Unsubscribe(ch)
}
