package rte

import (
	"fmt"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

// Signal represents the payload received from RTE.
type Signal struct {
	SignalType string            `json:"signal_type"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Power      float64           `json:"power"`
	Meta       map[string]string `json:"meta"`
}

// Validate checks that the signal payload is valid.
func (s Signal) Validate() error {
	if _, err := s.toSignalType(); err != nil {
		return err
	}
	if s.StartTime.IsZero() || s.EndTime.IsZero() {
		return fmt.Errorf("start_time and end_time required")
	}
	if !s.EndTime.After(s.StartTime) {
		return fmt.Errorf("end_time must be after start_time")
	}
	if s.Power == 0 {
		return fmt.Errorf("power must be non-zero")
	}
	return nil
}

func (s Signal) toSignalType() (model.SignalType, error) {
	switch s.SignalType {
	case "FCR":
		return model.SignalFCR, nil
	case "aFRR":
		return model.SignalAFRR, nil
	case "MA":
		return model.SignalMA, nil
	case "NEBEF":
		return model.SignalNEBEF, nil
	default:
		return 0, fmt.Errorf("unknown signal type: %s", s.SignalType)
	}
}

// ToFlexibility converts the RTE signal into a FlexibilitySignal used by the dispatch manager.
func (s Signal) ToFlexibility() (model.FlexibilitySignal, error) {
	st, err := s.toSignalType()
	if err != nil {
		return model.FlexibilitySignal{}, err
	}
	return model.FlexibilitySignal{
		Type:      st,
		PowerKW:   s.Power,
		Duration:  s.EndTime.Sub(s.StartTime),
		Timestamp: s.StartTime,
	}, nil
}
