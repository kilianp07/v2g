package events

import (
	"time"

	"github.com/kilianp07/v2g/model"
)

// AckEvent is published for each vehicle acknowledgment or error.
type AckEvent struct {
	VehicleID    string
	Signal       model.SignalType
	Acknowledged bool
	Err          error
	Latency      time.Duration
}
