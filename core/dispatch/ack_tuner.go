package dispatch

import (
	"errors"
	"sync"

	"github.com/kilianp07/v2g/core/mqtt"
)

// AckBasedTuner adjusts AvailabilityWeight of a SmartDispatcher based on ACK statistics.
// AckBasedTuner adjusts the AvailabilityWeight of a SmartDispatcher based on
// acknowledgment rates and timeout occurrences.
type AckBasedTuner struct {
	Dispatcher   *SmartDispatcher
	IncreaseStep float64
	DecreaseStep float64
	MaxWeight    float64
	MinWeight    float64
	Threshold    float64

	mu sync.Mutex
}

// NewAckBasedTuner returns an AckBasedTuner bound to the given dispatcher.
// DefaultAckThreshold defines the minimum acknowledgment rate required to
// trigger an increase of the AvailabilityWeight.
const DefaultAckThreshold = 0.9

// NewAckBasedTuner returns an AckBasedTuner with sane defaults. It returns nil
// if the dispatcher is nil.
func NewAckBasedTuner(d *SmartDispatcher) *AckBasedTuner {
	return NewAckBasedTunerWithConfig(d, 0.05, 0.05, 1, 0, DefaultAckThreshold)
}

// NewAckBasedTunerWithConfig creates an AckBasedTuner with custom parameters.
// If validation fails, the function returns nil.
func NewAckBasedTunerWithConfig(d *SmartDispatcher, increaseStep, decreaseStep, maxWeight, minWeight, threshold float64) *AckBasedTuner {
	if d == nil || increaseStep <= 0 || decreaseStep <= 0 || maxWeight < minWeight {
		return nil
	}
	return &AckBasedTuner{
		Dispatcher:   d,
		IncreaseStep: increaseStep,
		DecreaseStep: decreaseStep,
		MaxWeight:    maxWeight,
		MinWeight:    minWeight,
		Threshold:    threshold,
	}
}

// Tune modifies the dispatcher's AvailabilityWeight based on acknowledgment rate and timeouts.
func (t *AckBasedTuner) Tune(history []DispatchResult) {
	if t == nil || t.Dispatcher == nil || len(history) == 0 {
		return
	}

	var total, success float64
	var timeouts int
	for _, h := range history {
		for id := range h.Assignments {
			total++
			if h.Acknowledged[id] {
				success++
			}
			if err, ok := h.Errors[id]; ok && errors.Is(err, mqtt.ErrAckTimeout) {
				timeouts++
			}
		}
	}
	if total == 0 {
		return
	}

	rate := success / total
	var delta float64
	if rate >= t.Threshold {
		delta = t.IncreaseStep
	} else if timeouts > 0 {
		delta = -t.DecreaseStep
	}

	if delta != 0 {
		t.mu.Lock()
		t.Dispatcher.AvailabilityWeight += delta
		if t.Dispatcher.AvailabilityWeight > t.MaxWeight {
			t.Dispatcher.AvailabilityWeight = t.MaxWeight
		}
		if t.Dispatcher.AvailabilityWeight < t.MinWeight {
			t.Dispatcher.AvailabilityWeight = t.MinWeight
		}
		t.mu.Unlock()
	}
}
