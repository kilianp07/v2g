package rte

import (
	"testing"
	"time"
)

func TestSignalValidate(t *testing.T) {
	sig := Signal{
		SignalType: "FCR",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(15 * time.Minute),
		Power:      10,
	}
	if err := sig.Validate(); err != nil {
		t.Fatalf("valid signal rejected: %v", err)
	}

	bad := sig
	bad.SignalType = "foo"
	if err := bad.Validate(); err == nil {
		t.Errorf("invalid type not detected")
	}
	bad = sig
	bad.EndTime = bad.StartTime.Add(-time.Minute)
	if err := bad.Validate(); err == nil {
		t.Errorf("end before start not detected")
	}
}
