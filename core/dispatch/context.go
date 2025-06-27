package dispatch

import (
	"sync"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

// DispatchContext provides additional information used when scoring vehicles.
type DispatchContext struct {
	Signal             model.FlexibilitySignal
	Now                time.Time
	MarketPrice        float64
	ParticipationScore map[string]float64
	mu                 sync.RWMutex
}

// GetParticipation returns the participation score for the given vehicle in a
// thread-safe manner.
func (ctx *DispatchContext) GetParticipation(id string) float64 {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.ParticipationScore[id]
}

// SetParticipation safely updates the participation score for a vehicle.
func (ctx *DispatchContext) SetParticipation(id string, score float64) {
	ctx.mu.Lock()
	if ctx.ParticipationScore == nil {
		ctx.ParticipationScore = make(map[string]float64)
	}
	ctx.ParticipationScore[id] = score
	ctx.mu.Unlock()
}
