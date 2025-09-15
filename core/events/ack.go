package events

import (
	"time"

	"github.com/kilianp07/v2g/core/model"
)

// AckEvent is published for each vehicle acknowledgment or error.
type AckEvent struct {
	// OrderID identifies the dispatch order associated with this ACK.
	OrderID      string
	VehicleID    string
	Signal       model.SignalType
	Acknowledged bool
	Err          error
	Latency      time.Duration
}
