package dispatch

import (
	"fmt"
	"sync"
	"time"

	"github.com/kilianp07/v2g/logger"
	"github.com/kilianp07/v2g/metrics"
	"github.com/kilianp07/v2g/model"
	"github.com/kilianp07/v2g/mqtt"
)

type DispatchManager struct {
	filter     VehicleFilter
	dispatcher Dispatcher
	fallback   FallbackStrategy
	publisher  mqtt.Client
	ackTimeout time.Duration
	logger     logger.Logger
	metrics    metrics.MetricsSink
}

// NewDispatchManager creates a new manager.
// ackTimeout defines the maximum duration to wait for acknowledgments from vehicles.
// If ackTimeout is zero, a default of five seconds is used.
func NewDispatchManager(filter VehicleFilter, dispatcher Dispatcher, fallback FallbackStrategy, publisher mqtt.Client, ackTimeout time.Duration, log logger.Logger, sink metrics.MetricsSink) (*DispatchManager, error) {
	if filter == nil || dispatcher == nil || fallback == nil || publisher == nil {
		return nil, fmt.Errorf("dispatch: nil parameter provided to NewDispatchManager")
	}
	if ackTimeout <= 0 {
		ackTimeout = 5 * time.Second
	}
	if log == nil {
		log = logger.NopLogger{}
	}
	return &DispatchManager{
		filter:     filter,
		dispatcher: dispatcher,
		fallback:   fallback,
		publisher:  publisher,
		ackTimeout: ackTimeout,
		logger:     log,
		metrics:    sink,
	}, nil
}

// Dispatch runs the dispatch process.
func (m *DispatchManager) Dispatch(signal model.FlexibilitySignal, vehicles []model.Vehicle) DispatchResult {
	filtered := m.filter.Filter(vehicles, signal)
	assignments := m.dispatcher.Dispatch(filtered, signal)
	m.logger.Infof("dispatching %s to %d vehicles", signal.Type, len(filtered))

	result := DispatchResult{
		Assignments:  make(map[string]float64, len(assignments)),
		Errors:       make(map[string]error),
		Acknowledged: make(map[string]bool),
		Scores:       make(map[string]float64),
	}
	for id, p := range assignments {
		result.Assignments[id] = p
	}

	if sd, ok := m.dispatcher.(ScoringDispatcher); ok {
		for id, s := range sd.GetScores() {
			result.Scores[id] = s
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	update := func(id string, ack bool, err error) {
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.Errors[id] = err
		}
		result.Acknowledged[id] = err == nil && ack
	}
	for id, power := range result.Assignments {
		id := id
		power := power
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmdID, err := m.publisher.SendOrder(id, power)
			if err == nil {
				var ok bool
				ok, err = m.publisher.WaitForAck(cmdID, m.ackTimeout)
				update(id, ok, err)
			} else {
				update(id, false, err)
			}
		}()
	}
	wg.Wait()

	var failed []model.Vehicle
	for _, v := range filtered {
		if !result.Acknowledged[v.ID] {
			failed = append(failed, v)
		}
	}
	if len(failed) > 0 {
		m.logger.Warnf("%d vehicles failed, reallocating", len(failed))
		result.FallbackAssignments = m.fallback.Reallocate(failed, result.Assignments, signal)
	}
	result.Signal = signal
	if sd, ok := m.dispatcher.(*SmartDispatcher); ok {
		result.MarketPrice = sd.MarketPrice
	}
	if m.metrics != nil {
		var recs []metrics.DispatchResult
		for vid, p := range result.Assignments {
			recs = append(recs, metrics.DispatchResult{
				Signal:       result.Signal,
				VehicleID:    vid,
				PowerKW:      p,
				Score:        result.Scores[vid],
				MarketPrice:  result.MarketPrice,
				Acknowledged: result.Acknowledged[vid],
				DispatchTime: time.Now(),
			})
		}
		if err := m.metrics.RecordDispatchResult(recs); err != nil {
			m.logger.Errorf("metrics error: %v", err)
		}
	}
	return result
}
