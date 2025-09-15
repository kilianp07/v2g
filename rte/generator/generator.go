package generator

import (
	"context"
	"math/rand"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kilianp07/v2g/config"
	coreevents "github.com/kilianp07/v2g/core/events"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	coremodel "github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/internal/eventbus"
	"github.com/kilianp07/v2g/rte"
)

// Generator periodically emits synthetic RTE signals.
type Generator struct {
	cfg  config.RTEGeneratorConfig
	bus  eventbus.EventBus
	sink coremetrics.RTESignalRecorder
	mgr  rte.Manager
	log  logger.Logger
	rand *rand.Rand
	seq  int
}

var (
	signalsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "rte_generator_signals_total",
		Help: "Total RTE signals emitted",
	}, []string{"signal_type"})
	powerSum = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rte_generator_power_requested_kw_sum",
		Help: "Sum of requested power",
	})
	lastEmit = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rte_generator_last_emit_timestamp_seconds",
		Help: "Last emission time",
	})
	emitErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rte_generator_emit_errors_total",
		Help: "Errors while emitting signals",
	})
	intervalHist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "rte_generator_interval_seconds",
		Help:    "Interval between signals",
		Buckets: prometheus.DefBuckets,
	})
)

func init() {
	prometheus.MustRegister(signalsTotal, powerSum, lastEmit, emitErrors, intervalHist)
}

// New creates a new Generator.
func New(cfg config.RTEGeneratorConfig, mgr rte.Manager, bus eventbus.EventBus, sink coremetrics.RTESignalRecorder) *Generator {
	return &Generator{
		cfg:  cfg,
		bus:  bus,
		sink: sink,
		mgr:  mgr,
		log:  logger.New("rte-generator"),
		rand: rand.New(rand.NewSource(cfg.Seed)),
	}
}

// Start begins emitting signals until context cancellation.
func (g *Generator) Start(ctx context.Context) {
	for {
		interval := g.randomInterval()
		intervalHist.Observe(interval.Seconds())
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		sig := g.Generate(time.Now())
		if err := g.emit(sig); err != nil {
			emitErrors.Inc()
			g.log.Errorf("emit: %v", err)
		}
	}
}

// Generate produces a new flexibility signal at the given time.
func (g *Generator) Generate(now time.Time) coremodel.FlexibilitySignal {
	g.seq++
	typ := g.randomType()
	power := g.randomFloat(g.cfg.MinPowerKW, g.cfg.MaxPowerKW)
	duration := g.randomDuration(g.cfg.MinDurationSeconds, g.cfg.MaxDurationSeconds)
	return coremodel.FlexibilitySignal{
		Type:      typ,
		PowerKW:   power,
		Duration:  duration,
		Timestamp: now,
	}
}

func (g *Generator) emit(sig coremodel.FlexibilitySignal) error {
	g.log.Infof("signal %s power=%.2f duration=%s", sig.Type, sig.PowerKW, sig.Duration)
	g.log.Debugf("signal %s power=%.2f duration=%s", sig.Type, sig.PowerKW, sig.Duration)
	if g.mgr != nil {
		g.mgr.Dispatch(sig, nil)
	}
	if g.bus != nil {
		g.bus.Publish(coreevents.SignalEvent{Signal: sig})
	}
	if g.sink != nil {
		if err := g.sink.RecordRTESignal(coremetrics.RTESignalEvent{Signal: sig, Time: time.Now()}); err != nil {
			return err
		}
	}
	signalsTotal.WithLabelValues(sig.Type.String()).Inc()
	powerSum.Add(sig.PowerKW)
	lastEmit.Set(float64(time.Now().Unix()))
	return nil
}

func (g *Generator) randomType() coremodel.SignalType {
	if len(g.cfg.SignalTypes) == 0 {
		return coremodel.SignalFCR
	}
	s := g.cfg.SignalTypes[g.rand.Intn(len(g.cfg.SignalTypes))]
	switch s {
	case "FCR":
		return coremodel.SignalFCR
	case "aFRR_UP", "aFRR_DOWN", "aFRR":
		return coremodel.SignalAFRR
	case "DELESTAGE":
		return coremodel.SignalNEBEF
	default:
		return coremodel.SignalFCR
	}
}

func (g *Generator) randomFloat(min, max float64) float64 {
	if max <= min {
		return min
	}
	f := min + g.rand.Float64()*(max-min)
	j := 1 + (g.rand.Float64()*2-1)*g.cfg.JitterPct
	f *= j
	if f < min {
		f = min
	}
	if f > max {
		f = max
	}
	return f
}

func (g *Generator) randomDuration(min, max int) time.Duration {
	if max <= min {
		return time.Duration(min) * time.Second
	}
	sec := float64(min) + g.rand.Float64()*float64(max-min)
	j := 1 + (g.rand.Float64()*2-1)*g.cfg.JitterPct
	sec *= j
	if sec < float64(min) {
		sec = float64(min)
	}
	if sec > float64(max) {
		sec = float64(max)
	}
	return time.Duration(sec) * time.Second
}

func (g *Generator) randomInterval() time.Duration {
	return g.randomDuration(g.cfg.MinIntervalSeconds, g.cfg.MaxIntervalSeconds)
}
