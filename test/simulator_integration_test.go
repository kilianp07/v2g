//go:build !no_containers

package test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kilianp07/v2g/core/dispatch"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/metrics"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
	"github.com/kilianp07/v2g/test/util"
)

// syncBuffer is a thread-safe buffer for capturing command output
type syncBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (s *syncBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

type recordingSink struct {
	coremetrics.NopSink
	mu     sync.Mutex
	states int
	orders int
	acks   int
}

func (r *recordingSink) RecordVehicleState(ev coremetrics.VehicleStateEvent) error {
	r.mu.Lock()
	r.states++
	r.mu.Unlock()
	return nil
}

func (r *recordingSink) RecordDispatchOrder(ev coremetrics.DispatchOrderEvent) error {
	r.mu.Lock()
	r.orders++
	r.mu.Unlock()
	return nil
}

func (r *recordingSink) RecordDispatchAck(ev coremetrics.DispatchAckEvent) error {
	r.mu.Lock()
	r.acks++
	r.mu.Unlock()
	return nil
}

func (r *recordingSink) GetAcks() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.acks
}

func (r *recordingSink) GetOrders() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.orders
}

func TestSimulatorAndDispatcherIntegration(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}
	ctx := context.Background()
	broker, cleanup, err := util.StartMosquitto(ctx)
	if err != nil {
		t.Fatalf("start mosquitto: %v", err)
	}
	defer cleanup()

	// start simulator process
	simCtx, cancelSim := context.WithCancel(ctx)
	defer cancelSim()

	cmd, simOut := setupSimulatorCommand(simCtx, broker)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	defer cleanupSimulator(cancelSim, cmd, simOut, t)

	waitCtx, waitCancel := context.WithTimeout(ctx, 5*time.Second)
	defer waitCancel()
	if err := waitForSimulatorReady(waitCtx, broker); err != nil {
		t.Fatalf("simulator ready: %v", err)
	}
	vehicles := discoverVehicles(ctx, broker, t)
	recSink, reg, bus, collectorCancel := setupMetricsAndEventCollector(ctx, t)
	defer collectorCancel()
	defer bus.Close()

	dispatchSignalAndVerify(ctx, broker, vehicles, recSink, reg, bus, t)
}

func setupSimulatorCommand(simCtx context.Context, broker string) (*exec.Cmd, *syncBuffer) {
	cmd := exec.CommandContext(simCtx, "go", "run", "./simulator", "--broker="+broker, "--count=1", "--verbose", "--interval=1s")
	cmd.Dir = ".."

	// Utiliser un buffer thread-safe pour capturer la sortie
	var simOut syncBuffer
	cmd.Stdout = &simOut
	cmd.Stderr = &simOut

	return cmd, &simOut
}

func cleanupSimulator(cancelSim context.CancelFunc, cmd *exec.Cmd, simOut *syncBuffer, t *testing.T) {
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
}

func waitForSimulatorReady(ctx context.Context, broker string) error {
	discCfg := mqtt.Config{Broker: broker, ClientID: "ready-check"}
	disc, err := mqtt.NewPahoFleetDiscovery(discCfg, "v2g/fleet/discovery", "v2g/fleet/response/+", "hello")
	if err != nil {
		return err
	}
	defer func() {
		if err := disc.Close(); err != nil {
			fmt.Printf("close discovery: %v\n", err)
		}
	}()

	for {
		dctx, dcancel := context.WithTimeout(ctx, time.Second)
		vehicles, err := disc.Discover(dctx, time.Second)
		dcancel()
		if err == nil && len(vehicles) > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("simulator not ready: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func discoverVehicles(ctx context.Context, broker string, t *testing.T) []model.Vehicle {
	discCfg := mqtt.Config{Broker: broker, ClientID: "tester"}
	disc, err := mqtt.NewPahoFleetDiscovery(discCfg, "v2g/fleet/discovery", "v2g/fleet/response/+", "hello")
	if err != nil {
		t.Fatalf("discovery init: %v", err)
	}
	defer func() {
		if err := disc.Close(); err != nil {
			t.Logf("close discovery: %v", err)
		}
	}()

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

	return vehicles
}

func setupMetricsAndEventCollector(ctx context.Context, t *testing.T) (*recordingSink, *prometheus.Registry, *eventbus.Bus, context.CancelFunc) {
	reg := prometheus.NewRegistry()
	promSinkIf, err := metrics.NewPromSinkWithRegistry(coremetrics.Config{}, reg)
	if err != nil {
		t.Fatalf("prom sink: %v", err)
	}
	promSink := promSinkIf.(*metrics.PromSink)

	recSink := &recordingSink{}
	sink := coremetrics.NewMultiSink(promSink, recSink)

	bus := eventbus.New()
	// Créer un contexte pour le collector d'événements qui se ferme à la fin du test
	collectorCtx, collectorCancel := context.WithCancel(ctx)

	metrics.StartEventCollector(collectorCtx, bus, sink)

	return recSink, reg, bus, collectorCancel
}

func dispatchSignalAndVerify(ctx context.Context, broker string, vehicles []model.Vehicle, recSink *recordingSink, reg *prometheus.Registry, bus *eventbus.Bus, t *testing.T) {
	pub, err := mqtt.NewPahoClient(mqtt.Config{Broker: broker, ClientID: "dispatcher", AckTopic: "vehicle/+/ack"})
	if err != nil {
		t.Fatalf("mqtt client: %v", err)
	}

	mgr, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatch.EqualDispatcher{},
		dispatch.NoopFallback{},
		pub,
		time.Second,
		recSink,
		bus,
		nil,
		logger.NopLogger{},
		nil,
		nil,
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
	res := mgr.Dispatch(sig, vehicles)
	if len(res.Errors) > 0 {
		t.Fatalf("dispatch errors: %v", res.Errors)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	metricsTS := httptest.NewServer(mux)
	defer metricsTS.Close()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), util.MetricTimeout)
	defer waitCancel()
	metric := fmt.Sprintf(`dispatch_events_total{acknowledged="true",signal_type="FCR",vehicle_id="%s"} 1`, vehicles[0].ID)
	if err := util.WaitForMetric(waitCtx, metricsTS.URL+"/metrics", metric); err != nil {
		t.Errorf("metric wait: %v", err)
	}

	// Wait for asynchronous events to be processed by polling for acknowledgements
	for i := 0; i < 50; i++ {
		if recSink.GetAcks() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if recSink.GetAcks() == 0 {
		t.Errorf("no dispatch acks recorded")
	}
}
