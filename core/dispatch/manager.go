package dispatch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kilianp07/v2g/core/dispatch/logging"
	"github.com/kilianp07/v2g/core/events"
	"github.com/kilianp07/v2g/core/logger"
	"github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/core/mqtt"
	"github.com/kilianp07/v2g/core/prediction"
	vehiclestatus "github.com/kilianp07/v2g/core/vehiclestatus"
	"github.com/kilianp07/v2g/internal/eventbus"
)

type DispatchManager struct {
	filter       VehicleFilter
	dispatcher   Dispatcher
	lpDispatcher *LPDispatcher
	lpFirst      map[model.SignalType]bool
	fallback     FallbackStrategy
	publisher    mqtt.Client
	discovery    FleetDiscovery
	ackTimeout   time.Duration
	logger       logger.Logger
	metrics      metrics.MetricsSink
	bus          eventbus.EventBus
	tuner        LearningTuner
	prediction   prediction.PredictionEngine
	store        logging.LogStore
	statusStore  vehiclestatus.Store
	history      []DispatchResult
	mu           sync.Mutex
}

// SetLPFirst configures which signal types should try LP dispatch first.
func (m *DispatchManager) SetLPFirst(cfg map[model.SignalType]bool) {
	if cfg == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lpFirst = make(map[model.SignalType]bool, len(cfg))
	for k, v := range cfg {
		m.lpFirst[k] = v
	}
}

// SetLogStore configures the store used to persist dispatch logs.
func (m *DispatchManager) SetLogStore(store logging.LogStore) {
	m.mu.Lock()
	m.store = store
	m.mu.Unlock()
}

// SetStatusStore configures the store used to persist vehicle status information.
func (m *DispatchManager) SetStatusStore(store vehiclestatus.Store) {
	m.mu.Lock()
	m.statusStore = store
	m.mu.Unlock()
}

// dispatchStrategy selects the appropriate dispatcher based on configuration
// and falls back from LP to Smart on failure.
func (m *DispatchManager) dispatchStrategy(v []model.Vehicle, s model.FlexibilitySignal) (map[string]float64, Dispatcher) {
	m.mu.Lock()
	lpFirst := m.lpFirst[s.Type]
	m.mu.Unlock()

	if lpFirst && m.lpDispatcher != nil {
		if m.bus != nil {
			m.bus.Publish(events.StrategyEvent{Signal: s.Type, Action: "lp_attempt"})
		}
		m.logger.Debugf("trying LP dispatch for %s", s.Type)
		asn, err := m.lpDispatcher.DispatchStrict(v, s)
		if err == nil {
			return asn, m.lpDispatcher
		}
		if m.bus != nil {
			m.bus.Publish(events.StrategyEvent{Signal: s.Type, Action: "lp_failure", Err: err})
		}
		m.logger.Warnf("LP dispatch failed: %v", err)
		assignments := m.dispatcher.Dispatch(v, s)
		if m.bus != nil {
			m.bus.Publish(events.StrategyEvent{Signal: s.Type, Action: "smart_fallback"})
		}
		return assignments, m.dispatcher
	}
	return m.dispatcher.Dispatch(v, s), m.dispatcher
}

// Close releases resources held by the manager.
func (m *DispatchManager) Close() error {
	if m.discovery != nil {
		if err := m.discovery.Close(); err != nil {
			return err
		}
	}
	if m.bus != nil {
		m.bus.Close()
	}
	if m.store != nil {
		_ = m.store.Close()
	}
	return nil
}

// Run processes incoming flexibility signals until the context is canceled.
// For each signal received on the channel, Dispatch is invoked. If a
// FleetDiscovery is configured, the vehicles are discovered before each
// dispatch.
func (m *DispatchManager) Run(ctx context.Context, signals <-chan model.FlexibilitySignal) {
	for {
		select {
		case sig := <-signals:
			m.Dispatch(sig, nil)
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
		mqttFailure.Inc()
		return false, time.Since(start), err
	}
	mqttSuccess.Inc()
	ack, err := m.publisher.WaitForAck(cmdID, m.ackTimeout)
	return ack, time.Since(start), err
}

// NewDispatchManager creates a new manager.
// ackTimeout defines the maximum duration to wait for acknowledgments from vehicles.
// If ackTimeout is zero, a default of five seconds is used.
func NewDispatchManager(filter VehicleFilter, dispatcher Dispatcher, fallback FallbackStrategy, publisher mqtt.Client, ackTimeout time.Duration, sink metrics.MetricsSink, bus eventbus.EventBus, disc FleetDiscovery, log logger.Logger, tuner LearningTuner, pred prediction.PredictionEngine) (*DispatchManager, error) {
	if filter == nil || dispatcher == nil || fallback == nil || publisher == nil {
		return nil, fmt.Errorf("dispatch: nil parameter provided to NewDispatchManager")
	}
	if ackTimeout <= 0 {
		ackTimeout = 5 * time.Second
	}

	mgr := &DispatchManager{
		filter:     filter,
		dispatcher: dispatcher,
		fallback:   fallback,
		publisher:  publisher,
		discovery:  disc,
		ackTimeout: ackTimeout,
		logger:     log,
		metrics:    sink,
		bus:        bus,
		tuner:      tuner,
		prediction: pred,
		lpFirst:    make(map[model.SignalType]bool),
	}
	switch d := dispatcher.(type) {
	case *LPDispatcher:
		mgr.lpDispatcher = d
	case *SmartDispatcher:
		lp := NewLPDispatcher()
		lp.SmartDispatcher = *d
		mgr.lpDispatcher = &lp
	}
	return mgr, nil
}

// Dispatch runs the dispatch process.
//
//gocyclo:ignore
func (m *DispatchManager) Dispatch(signal model.FlexibilitySignal, vehicles []model.Vehicle) DispatchResult {
	if len(vehicles) == 0 && m.discovery != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if vs, err := m.discovery.Discover(ctx, time.Second); err == nil {
			vehicles = vs
			if fr, ok := m.metrics.(metrics.FleetSizeRecorder); ok {
				if err := fr.RecordFleetSize(len(vs)); err != nil {
					m.logger.Errorf("fleet size metrics error: %v", err)
				}
			}
			m.logger.Infof("discovered %d vehicles", len(vs))
		} else {
			m.logger.Errorf("fleet discovery failed: %v", err)
		}
	}
	filtered := m.filter.Filter(vehicles, signal)
	if m.prediction != nil {
		horizon := signal.Duration
		if horizon <= 0 {
			horizon = time.Hour
		}
		for i, v := range filtered {
			filtered[i].AvailabilityProb = m.prediction.PredictAvailability(v.ID, signal.Timestamp.Add(horizon))
			if fc := m.prediction.ForecastSoC(v.ID, horizon); len(fc) > 0 {
				filtered[i].SoC = fc[len(fc)-1]
			}
		}
	}
	if va, ok := m.fallback.(VehicleAwareFallback); ok {
		va.SetVehicles(filtered)
	}
	if m.bus != nil {
		m.bus.Publish(events.SignalEvent{Signal: signal})
	}
	assignments, used := m.dispatchStrategy(filtered, signal)
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

	if sd, ok := used.(ScoringDispatcher); ok {
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
	if mp, ok := used.(MarketPriceProvider); ok {
		result.MarketPrice = mp.GetMarketPrice()
	}
	m.recordMetrics(result, latencies, lr, recordLatency)
	m.mu.Lock()
	m.history = append(m.history, result)
	hist := append([]DispatchResult(nil), m.history...)
	m.mu.Unlock()
	if m.store != nil {
		vids := make([]string, 0, len(filtered))
		for _, v := range filtered {
			vids = append(vids, v.ID)
		}
		lr := logging.Result{
			Assignments:         result.Assignments,
			FallbackAssignments: result.FallbackAssignments,
			Errors:              map[string]string{},
			Acknowledged:        result.Acknowledged,
			Signal:              result.Signal,
			MarketPrice:         result.MarketPrice,
			Scores:              result.Scores,
		}
		for id, err := range result.Errors {
			if err != nil {
				lr.Errors[id] = err.Error()
			}
		}
		_ = m.store.Append(context.Background(), logging.LogRecord{
			Timestamp:        time.Now(),
			Signal:           signal,
			TargetPower:      signal.PowerKW,
			VehiclesSelected: vids,
			Response:         lr,
		})
	}
	if m.statusStore != nil {
		dec := vehiclestatus.LastDispatch{
			SignalType:       signal.Type.String(),
			TargetPower:      signal.PowerKW,
			VehiclesSelected: make([]string, 0, len(result.Assignments)),
			Timestamp:        signal.Timestamp,
		}
		for id := range result.Assignments {
			dec.VehiclesSelected = append(dec.VehiclesSelected, id)
		}
		for id := range result.Assignments {
			m.statusStore.RecordDispatch(id, dec)
		}
	}
	if m.tuner != nil {
		m.tuner.Tune(hist)
	}
	return result
}

// dispatchAssignments publishes the orders concurrently and records acknowledgments.
func (m *DispatchManager) dispatchAssignments(res *DispatchResult, signal model.FlexibilitySignal, recordLatency bool) []metrics.DispatchLatency {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		lat      []metrics.DispatchLatency
		ackCount int
	)
	update := func(id string, ack bool, err error, dur time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			res.Errors[id] = err
		}
		res.Acknowledged[id] = err == nil && ack
		vehiclesDispatched.WithLabelValues(signal.Type.String()).Inc()
		dispatchLatency.WithLabelValues(signal.Type.String()).Observe(dur.Seconds())
		if m.bus != nil {
			m.bus.Publish(events.AckEvent{
				VehicleID:    id,
				Signal:       signal.Type,
				Acknowledged: ack && err == nil,
				Err:          err,
				Latency:      dur,
			})
		}
		if recordLatency {
			lat = append(lat, metrics.DispatchLatency{
				VehicleID:    id,
				Signal:       signal.Type,
				Acknowledged: err == nil && ack,
				Latency:      dur,
			})
		}
		if err == nil && ack {
			ackCount++
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
	if total := len(res.Assignments); total > 0 {
		ackRate.WithLabelValues(signal.Type.String()).Set(float64(ackCount) / float64(total))
	}
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
