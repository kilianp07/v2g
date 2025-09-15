package generator

import (
	"context"
	"testing"
	"time"

	"github.com/kilianp07/v2g/config"
	dispatch "github.com/kilianp07/v2g/core/dispatch"
	coreevents "github.com/kilianp07/v2g/core/events"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	coremodel "github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/internal/eventbus"
)

type recSink struct{ ev coremetrics.RTESignalEvent }

func (r *recSink) RecordRTESignal(ev coremetrics.RTESignalEvent) error { r.ev = ev; return nil }

func TestGeneratorDeterministic(t *testing.T) {
	cfg := config.RTEGeneratorConfig{
		Enabled:            true,
		MinIntervalSeconds: 1,
		MaxIntervalSeconds: 1,
		MinDurationSeconds: 60,
		MaxDurationSeconds: 60,
		MinPowerKW:         5,
		MaxPowerKW:         5,
		SignalTypes:        []string{"FCR"},
		Seed:               42,
	}
	bus := eventbus.New()
	g1 := New(cfg, nil, bus, &recSink{})
	g2 := New(cfg, nil, bus, &recSink{})
	now := time.Unix(0, 0)
	s1a := g1.Generate(now)
	s1b := g2.Generate(now)
	if s1a != s1b {
		t.Fatalf("expected deterministic generation")
	}
	s2a := g1.Generate(now)
	s2b := g2.Generate(now)
	if s2a != s2b {
		t.Fatalf("expected same second signal")
	}
}

func TestGeneratorPublish(t *testing.T) {
	cfg := config.RTEGeneratorConfig{
		Enabled:            true,
		MinIntervalSeconds: 0,
		MaxIntervalSeconds: 0,
		MinDurationSeconds: 1,
		MaxDurationSeconds: 1,
		MinPowerKW:         1,
		MaxPowerKW:         1,
		SignalTypes:        []string{"FCR"},
		Seed:               1,
	}
	bus := eventbus.New()
	sink := &recSink{}
	g := New(cfg, nil, bus, sink)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := bus.Subscribe()
	go g.Start(ctx)
	select {
	case e := <-ch:
		if _, ok := e.(coreevents.SignalEvent); !ok {
			t.Fatalf("unexpected event %T", e)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("no event received")
	}
	if sink.ev.Signal.PowerKW == 0 {
		t.Fatalf("expected sink to record signal")
	}
}

func TestGeneratorBounds(t *testing.T) {
	cfg := config.RTEGeneratorConfig{
		Enabled:            true,
		MinIntervalSeconds: 0,
		MaxIntervalSeconds: 0,
		MinDurationSeconds: 10,
		MaxDurationSeconds: 20,
		MinPowerKW:         5,
		MaxPowerKW:         10,
		SignalTypes:        []string{"FCR"},
		Seed:               3,
		JitterPct:          0.2,
	}
	g := New(cfg, nil, nil, &recSink{})
	now := time.Unix(0, 0)
	for i := 0; i < 50; i++ {
		s := g.Generate(now)
		if s.PowerKW < cfg.MinPowerKW || s.PowerKW > cfg.MaxPowerKW {
			t.Fatalf("power out of bounds: %f", s.PowerKW)
		}
		if s.Duration < time.Duration(cfg.MinDurationSeconds)*time.Second || s.Duration > time.Duration(cfg.MaxDurationSeconds)*time.Second {
			t.Fatalf("duration out of bounds: %s", s.Duration)
		}
	}
}

type mockMgr struct{ sig coremodel.FlexibilitySignal }

func (m *mockMgr) Dispatch(s coremodel.FlexibilitySignal, _ []coremodel.Vehicle) dispatch.DispatchResult {
	m.sig = s
	return dispatch.DispatchResult{}
}

func TestGeneratorDispatches(t *testing.T) {
	cfg := config.RTEGeneratorConfig{Enabled: true, MinIntervalSeconds: 0, MaxIntervalSeconds: 0, MinDurationSeconds: 1, MaxDurationSeconds: 1, MinPowerKW: 1, MaxPowerKW: 1, SignalTypes: []string{"FCR"}}
	bus := eventbus.New()
	m := &mockMgr{}
	g := New(cfg, m, bus, &recSink{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Start(ctx)
	time.Sleep(10 * time.Millisecond)
	if m.sig.PowerKW == 0 {
		t.Fatalf("expected dispatch to be called")
	}
}
