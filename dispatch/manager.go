package dispatch

import (
	"sync"
	"time"

	"github.com/kilianp07/v2g/model"
	"github.com/kilianp07/v2g/mqtt"
)

type DispatchManager struct {
	filter     VehicleFilter
	dispatcher Dispatcher
	fallback   FallbackStrategy
	publisher  mqtt.Client
	mu         sync.Mutex
}

// NewDispatchManager creates a new manager.
func NewDispatchManager(filter VehicleFilter, dispatcher Dispatcher, fallback FallbackStrategy, publisher mqtt.Client) *DispatchManager {
	if filter == nil || dispatcher == nil || fallback == nil || publisher == nil {
		panic("dispatch: nil parameter provided to NewDispatchManager")
	}
	return &DispatchManager{filter: filter, dispatcher: dispatcher, fallback: fallback, publisher: publisher}
}

// Dispatch runs the dispatch process.
func (m *DispatchManager) Dispatch(signal model.FlexibilitySignal, vehicles []model.Vehicle) DispatchResult {
	m.mu.Lock()

	filtered := m.filter.Filter(vehicles, signal)
	assignments := m.dispatcher.Dispatch(filtered, signal)

	result := DispatchResult{
		Assignments:  make(map[string]float64, len(assignments)),
		Errors:       make(map[string]error),
		Acknowledged: make(map[string]bool),
	}
	for id, p := range assignments {
		result.Assignments[id] = p
	}

	for id, power := range result.Assignments {
		cmdID, err := m.publisher.SendOrder(id, power)
		if err != nil {
			result.Errors[id] = err
			result.Acknowledged[id] = false
			continue
		}
		if ok, err := m.publisher.WaitForAck(cmdID, 5*time.Second); err != nil || !ok {
			if err != nil {
				result.Errors[id] = err
			}
			result.Acknowledged[id] = false
		} else {
			result.Acknowledged[id] = true
		}
	}

	var failed []model.Vehicle
	for _, v := range filtered {
		if !result.Acknowledged[v.ID] {
			failed = append(failed, v)
		}
	}
	if len(failed) > 0 {
		result.FallbackAssignments = m.fallback.Reallocate(failed, result.Assignments, signal)
	}

	m.mu.Unlock()
	return result
}
