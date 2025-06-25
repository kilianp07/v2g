package eventbus

import "testing"

func TestBusPublishSubscribe(t *testing.T) {
	bus := New()
	ch := bus.Subscribe()
	bus.Publish("hello")
	v := <-ch
	if v != "hello" {
		t.Fatalf("expected hello got %v", v)
	}
	bus.Unsubscribe(ch)
}

func TestBusClose(t *testing.T) {
	bus := New()
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

func TestBusUnsubscribeAfterClose(t *testing.T) {
	bus := New()
	ch := bus.Subscribe()
	bus.Close()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on Unsubscribe after Close: %v", r)
		}
	}()
	bus.Unsubscribe(ch)
}
