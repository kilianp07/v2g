package dispatch

import (
	"time"

	"github.com/kilianp07/v2g/model"
)

// DispatchContext provides additional information used when scoring vehicles.
type DispatchContext struct {
	Signal             model.FlexibilitySignal
	Now                time.Time
	MarketPrice        float64
	ParticipationScore map[string]float64
}
