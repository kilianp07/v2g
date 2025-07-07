package metrics

import "testing"

// TestMultiSink ensures events are forwarded to all sinks.

type recordSink struct {
	count int
}

func (r *recordSink) RecordDispatchResult([]DispatchResult) error {
	r.count++
	return nil
}

func (r *recordSink) RecordDispatchLatency([]DispatchLatency) error {
	r.count++
	return nil
}

func TestMultiSink(t *testing.T) {
	s1 := &recordSink{}
	s2 := &recordSink{}
	m := NewMultiSink(s1, s2)
	if err := m.RecordDispatchResult(nil); err != nil {
		t.Fatalf("record result: %v", err)
	}
	if err := m.RecordDispatchLatency(nil); err != nil {
		t.Fatalf("record latency: %v", err)
	}
	if s1.count != 2 || s2.count != 2 {
		t.Fatalf("results not forwarded")
	}
}
