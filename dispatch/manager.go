package dispatch

import (
	"context"
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

// Run processes incoming flexibility signals until the context is canceled.
// For each signal received on the channel, Dispatch is invoked with the
// provided list of vehicles.
func (m *DispatchManager) Run(ctx context.Context, signals <-chan model.FlexibilitySignal, vehicles []model.Vehicle) {
	for {
		select {
		case sig := <-signals:
			m.Dispatch(sig, vehicles)
		case <-ctx.Done():
			return
		}
	}
}

// sendAndWait sends the command and waits for an acknowledgment while measuring
// the latency.

func (m *DispatchManager) sendAndWait(id string, power float64) (bool, time.Duration, error) {
	start := time.Now()
	cmdID, err := m.publisher.SendOrder(id, power)
	if err != nil {
		return false, time.Since(start), err
	}
	ack, err := m.publisher.WaitForAck(cmdID, m.ackTimeout)
	return ack, time.Since(start), err
}

// NewDispatchManager creates a new manager.
// ackTimeout defines the maximum duration to wait for acknowledgments from vehicles.
// If ackTimeout is zero, a default of five seconds is used.
func NewDispatchManager(filter VehicleFilter, dispatcher Dispatcher, fallback FallbackStrategy, publisher mqtt.Client, ackTimeout time.Duration, sink metrics.MetricsSink) (*DispatchManager, error) {
	if filter == nil || dispatcher == nil || fallback == nil || publisher == nil {
		return nil, fmt.Errorf("dispatch: nil parameter provided to NewDispatchManager")
	}
	if ackTimeout <= 0 {
		ackTimeout = 5 * time.Second
	}

	return &DispatchManager{
		filter:     filter,
		dispatcher: dispatcher,
		fallback:   fallback,
		publisher:  publisher,
		ackTimeout: ackTimeout,
		logger:     logger.New("dispatch"),
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

	lr, recordLatency := m.metrics.(metrics.LatencyRecorder)
	latencies := m.dispatchAssignments(&result, signal, recordLatency)

	failed := m.unacknowledged(filtered, result.Acknowledged)
	if len(failed) > 0 {
		m.logger.Warnf("%d vehicles failed, reallocating", len(failed))
		result.FallbackAssignments = m.fallback.Reallocate(failed, result.Assignments, signal)
	}

	result.Signal = signal
	if mp, ok := m.dispatcher.(MarketPriceProvider); ok {
		result.MarketPrice = mp.GetMarketPrice()
	}
	m.recordMetrics(result, latencies, lr, recordLatency)
	return result
}

// dispatchAssignments publishes the orders concurrently and records acknowledgments.
func (m *DispatchManager) dispatchAssignments(res *DispatchResult, signal model.FlexibilitySignal, recordLatency bool) []metrics.DispatchLatency {
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		lat []metrics.DispatchLatency
	)
	update := func(id string, ack bool, err error, dur time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			res.Errors[id] = err
		}
		res.Acknowledged[id] = err == nil && ack
		if recordLatency {
			lat = append(lat, metrics.DispatchLatency{
				VehicleID:    id,
				Signal:       signal.Type,
				Acknowledged: err == nil && ack,
				Latency:      dur,
			})
		}
	}
	for id, power := range res.Assignments {
		wg.Add(1)
		go func(id string, p float64) {
			defer wg.Done()
			ack, d, err := m.sendAndWait(id, p)
			update(id, ack, err, d)
		}(id, power)
	}
	wg.Wait()
	return lat
}

// unacknowledged returns the subset of vehicles that did not acknowledge.
func (m *DispatchManager) unacknowledged(all []model.Vehicle, acks map[string]bool) []model.Vehicle {
	var failed []model.Vehicle
	for _, v := range all {
		if !acks[v.ID] {
			failed = append(failed, v)
		}
	}
	return failed
}

// recordMetrics persists dispatch metrics if a sink is configured.
func (m *DispatchManager) recordMetrics(res DispatchResult, lat []metrics.DispatchLatency, lr metrics.LatencyRecorder, hasLatency bool) {
	if m.metrics == nil {
		return
	}
	var recs []metrics.DispatchResult
	for vid, p := range res.Assignments {
		recs = append(recs, metrics.DispatchResult{
			Signal:       res.Signal,
			VehicleID:    vid,
			PowerKW:      p,
			Score:        res.Scores[vid],
			MarketPrice:  res.MarketPrice,
			Acknowledged: res.Acknowledged[vid],
			DispatchTime: res.Signal.Timestamp,
		})
	}
	if err := m.metrics.RecordDispatchResult(recs); err != nil {
		m.logger.Errorf("metrics error: %v", err)
	}
	if hasLatency && lr != nil {
		if err := lr.RecordDispatchLatency(lat); err != nil {
			m.logger.Errorf("latency metrics error: %v", err)
		}
	}
}
