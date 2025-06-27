package events

import "github.com/kilianp07/v2g/core/model"

// SignalEvent is published when a new flexibility signal is processed.
type SignalEvent struct {
	Signal model.FlexibilitySignal
}
