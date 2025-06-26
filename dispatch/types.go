package dispatch

import (
	"context"
	"time"

	"github.com/kilianp07/v2g/model"
)

// DispatchResult contains the result of a dispatch operation.
type DispatchResult struct {
	Assignments         map[string]float64 // initial allocation per vehicle
	FallbackAssignments map[string]float64 // reallocation if some vehicles fail
	Errors              map[string]error
	Acknowledged        map[string]bool
	Signal              model.FlexibilitySignal
	MarketPrice         float64
	Scores              map[string]float64
}

// Dispatcher defines how power is distributed between vehicles.
type Dispatcher interface {
	Dispatch(vehicles []model.Vehicle, signal model.FlexibilitySignal) map[string]float64
}

// VehicleFilter filters vehicles depending on the signal and their status.
type VehicleFilter interface {
	Filter(vehicles []model.Vehicle, signal model.FlexibilitySignal) []model.Vehicle
}

// FallbackStrategy reallocates power in case of failures.
type FallbackStrategy interface {
	Reallocate(failed []model.Vehicle, current map[string]float64, signal model.FlexibilitySignal) map[string]float64
}

// ScoringDispatcher optionally exposes per-vehicle scores after dispatch.
type ScoringDispatcher interface {
	GetScores() map[string]float64
}

// MarketPriceProvider exposes the current market price used by the dispatcher.
type MarketPriceProvider interface {
	GetMarketPrice() float64
}

// FleetDiscovery retrieves the current list of available vehicles.
// Discover should return within the provided timeout and must be non-blocking.
type FleetDiscovery interface {
	Discover(ctx context.Context, timeout time.Duration) ([]model.Vehicle, error)
	Close() error
}
