package dispatch

import "strings"

// AckBasedTuner adjusts AvailabilityWeight of a SmartDispatcher based on ACK statistics.
type AckBasedTuner struct {
	Dispatcher   *SmartDispatcher
	IncreaseStep float64
	DecreaseStep float64
	MaxWeight    float64
	MinWeight    float64
}

// NewAckBasedTuner returns an AckBasedTuner bound to the given dispatcher.
func NewAckBasedTuner(d *SmartDispatcher) *AckBasedTuner {
	return &AckBasedTuner{
		Dispatcher:   d,
		IncreaseStep: 0.05,
		DecreaseStep: 0.05,
		MaxWeight:    1,
		MinWeight:    0,
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
			if err, ok := h.Errors[id]; ok && strings.Contains(err.Error(), "timeout") {
				timeouts++
			}
		}
	}
	if total == 0 {
		return
	}
	rate := success / total
	if rate > 0.9 {
		t.Dispatcher.AvailabilityWeight += t.IncreaseStep
	}
	if timeouts > 0 {
		t.Dispatcher.AvailabilityWeight -= t.DecreaseStep
	}
	if t.Dispatcher.AvailabilityWeight > t.MaxWeight {
		t.Dispatcher.AvailabilityWeight = t.MaxWeight
	}
	if t.Dispatcher.AvailabilityWeight < t.MinWeight {
		t.Dispatcher.AvailabilityWeight = t.MinWeight
	}
}
