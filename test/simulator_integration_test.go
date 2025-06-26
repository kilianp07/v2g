package test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kilianp07/v2g/dispatch"
	"github.com/kilianp07/v2g/internal/eventbus"
	"github.com/kilianp07/v2g/metrics"
	"github.com/kilianp07/v2g/model"
	"github.com/kilianp07/v2g/mqtt"
)

type recordingSink struct {
	metrics.NopSink
	mu     sync.Mutex
	states int
	orders int
	acks   int
}

func (r *recordingSink) RecordVehicleState(ev metrics.VehicleStateEvent) error {
	r.mu.Lock()
	r.states++
	r.mu.Unlock()
	return nil
}

func (r *recordingSink) RecordDispatchOrder(ev metrics.DispatchOrderEvent) error {
	r.mu.Lock()
	r.orders++
	r.mu.Unlock()
	return nil
}

func (r *recordingSink) RecordDispatchAck(ev metrics.DispatchAckEvent) error {
	r.mu.Lock()
	r.acks++
	r.mu.Unlock()
	return nil
}

func TestSimulatorAndDispatcherIntegration(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}
	ctx := context.Background()
	cont, broker := startMosquitto(ctx, t)
	defer func() { _ = cont.Terminate(ctx) }()

	// start simulator process
	simCtx, cancelSim := context.WithCancel(ctx)
	defer cancelSim()

	cmd := exec.CommandContext(simCtx, "go", "run", "./simulator", "--broker="+broker, "--count=1", "--verbose", "--interval=1s")
	cmd.Dir = ".."

	var simOut bytes.Buffer
	cmd.Stdout = &simOut
	cmd.Stderr = &simOut

	if err := cmd.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}

	defer func() {
		cancelSim()
		done := make(chan error)
		go func() { done <- cmd.Wait() }()
		select {
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			t.Logf("simulator killed due to timeout. Output:\n%s", simOut.String())
		case err := <-done:
			if err != nil {
				t.Logf("simulator exited with error: %v\nOutput:\n%s", err, simOut.String())
			}
		}
	}()

	time.Sleep(3 * time.Second) // Allow simulator to connect and subscribe

	discCfg := mqtt.Config{Broker: broker, ClientID: "tester"}
	disc, err := mqtt.NewPahoFleetDiscovery(discCfg, "v2g/fleet/discovery", "v2g/fleet/response/+", "hello")
	if err != nil {
		t.Fatalf("discovery init: %v", err)
	}

	var vehicles []model.Vehicle
	for i := 0; i < 5; i++ {
		dctx, dcancel := context.WithTimeout(ctx, 2*time.Second)
		vehicles, err = disc.Discover(dctx, 1*time.Second)
		dcancel()
		if err == nil && len(vehicles) > 0 {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(vehicles) == 0 {
		t.Fatal("expected at least 1 vehicle discovered")
	}
	t.Logf("discovered %d vehicles", len(vehicles))

	reg := prometheus.NewRegistry()
	promSinkIf, err := metrics.NewPromSinkWithRegistry(metrics.Config{}, reg)
	if err != nil {
		t.Fatalf("prom sink: %v", err)
	}
	promSink := promSinkIf.(*metrics.PromSink)

	recSink := &recordingSink{}
	sink := metrics.NewMultiSink(promSink, recSink)

	pub, err := mqtt.NewPahoClient(mqtt.Config{Broker: broker, ClientID: "dispatcher", AckTopic: "vehicle/+/ack"})
	if err != nil {
		t.Fatalf("mqtt client: %v", err)
	}

	bus := eventbus.New()
	metrics.StartEventCollector(ctx, bus, sink)

	mgr, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatch.EqualDispatcher{},
		dispatch.NoopFallback{},
		pub,
		time.Second,
		sink,
		bus,
		disc,
	)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	sig := model.FlexibilitySignal{
		Type:      model.SignalFCR,
		PowerKW:   5,
		Duration:  time.Minute,
		Timestamp: time.Now(),
	}
	res := mgr.Dispatch(sig, nil)
	if len(res.Errors) > 0 {
		t.Fatalf("dispatch errors: %v", res.Errors)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	metricsTS := httptest.NewServer(mux)
	defer metricsTS.Close()

	if err := waitForMetric(metricsTS.URL+"/metrics", `dispatch_events_total{acknowledged="true",signal_type="FCR",vehicle_id="veh001"} 1`, 10*time.Second); err != nil {
		t.Errorf("metric wait: %v", err)
	}

	if recSink.acks == 0 {
		t.Errorf("no dispatch acks recorded")
	}

	if err := disc.Close(); err != nil {
		t.Logf("close discovery: %v", err)
	}
}
