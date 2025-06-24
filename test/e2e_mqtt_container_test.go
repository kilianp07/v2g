package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/dispatch"
	"github.com/kilianp07/v2g/logger"
	"github.com/kilianp07/v2g/metrics"
	"github.com/kilianp07/v2g/model"
	"github.com/kilianp07/v2g/mqtt"
	"github.com/kilianp07/v2g/rte"
)

func waitForMQTTReady(broker string, timeout time.Duration) error {
	opts := paho.NewClientOptions().AddBroker(broker).SetClientID("probe")
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		cli := paho.NewClient(opts)
		token := cli.Connect()
		token.Wait()
		if token.Error() == nil {
			cli.Disconnect(100)
			return nil
		}
		lastErr = token.Error()
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timeout waiting for broker")
	}
	return lastErr
}

type containerWrapper struct {
	mgr      *dispatch.DispatchManager
	vehicles []model.Vehicle
}

func (c containerWrapper) Dispatch(sig model.FlexibilitySignal, _ []model.Vehicle) dispatch.DispatchResult {
	return c.mgr.Dispatch(sig, c.vehicles)
}

func startMosquitto(ctx context.Context, t *testing.T) (tc.Container, string) {
	t.Helper()
	conf := `listener 1883
allow_anonymous true
persistence false
log_dest stdout
log_type error
log_type warning
log_type notice
log_type information
connection_messages true
log_timestamp true
`
	path := "./testdata/mosquitto.conf"
	if err := os.MkdirAll("./testdata", 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(conf), 0644); err != nil {
		t.Fatalf("write conf: %v", err)
	}

	req := tc.ContainerRequest{
		Image:        "eclipse-mosquitto:2.0",
		ExposedPorts: []string{"1883/tcp"},
		WaitingFor:   wait.ForListeningPort("1883/tcp"),
		Files: []tc.ContainerFile{
			{
				HostFilePath:      path,
				ContainerFilePath: "/mosquitto/config/mosquitto.conf",
				FileMode:          0644,
			},
		},
	}
	cont, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatalf("container start: %v", err)
	}
	host, err := cont.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	port, err := cont.MappedPort(ctx, "1883")
	if err != nil {
		t.Fatalf("port: %v", err)
	}
	broker := fmt.Sprintf("tcp://%s:%s", host, port.Port())
	addr := net.JoinHostPort(host, port.Port())
	if err := waitForMQTTReady(broker, 5*time.Second); err != nil {
		t.Logf("mosquitto not ready at %s: %v", addr, err)
		t.Skip("Mosquitto not ready after retries")
	}
	return cont, broker
}

func connectAckClient(broker string, t *testing.T) paho.Client {
	t.Helper()
	ackOpts := paho.NewClientOptions().AddBroker(broker).SetClientID("ack-sim")
	ackCli := paho.NewClient(ackOpts)
	var connErr error
	time.Sleep(250 * time.Millisecond)
	for i := 0; i < 5; i++ {
		token := ackCli.Connect()
		token.Wait()
		connErr = token.Error()
		if connErr == nil {
			break
		}
		t.Logf("ack connect attempt %d to %s: %v", i+1, broker, connErr)
		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
	}
	if connErr != nil {
		t.Logf("ack connect failed to %s: %v", broker, connErr)
		t.Skip("Mosquitto not ready after retries")
	}
	if token := ackCli.Subscribe("vehicle/veh1/command", 0, func(_ paho.Client, m paho.Message) {
		var cmd struct {
			CommandID string `json:"command_id"`
		}
		_ = json.Unmarshal(m.Payload(), &cmd)
		payload, _ := json.Marshal(map[string]string{"command_id": cmd.CommandID})
		ackCli.Publish("vehicle/veh1/ack", 0, false, payload)
	}); token.Wait() && token.Error() != nil {
		t.Fatalf("subscribe: %v", token.Error())
	}
	return ackCli
}

func TestSignalDispatchWithMQTTContainer(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}
	ctx := context.Background()

	cont, broker := startMosquitto(ctx, t)
	defer func() { _ = cont.Terminate(ctx) }()

	ackCli := connectAckClient(broker, t)
	defer ackCli.Disconnect(100)

	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	sinkIf, err := metrics.NewPromSinkWithRegistry(metrics.Config{}, reg)
	if err != nil {
		t.Fatalf("prom sink: %v", err)
	}
	sink := sinkIf.(*metrics.PromSink)

	pub, err := mqtt.NewPahoClient(mqtt.Config{
		Broker:   broker,
		ClientID: "dispatcher",
		AckTopic: "vehicle/+/ack",
		Logger:   logger.New("test"),
	})
	if err != nil {
		t.Fatalf("mqtt client: %v", err)
	}

	mgr, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatch.EqualDispatcher{},
		dispatch.NoopFallback{},
		pub,
		time.Second,
		logger.New("test"),
		sink,
	)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	vehicles := []model.Vehicle{{ID: "veh1", SoC: 0.8, IsV2G: true, Available: true, MaxPower: 40, BatteryKWh: 50}}
	wrapper := containerWrapper{mgr: mgr, vehicles: vehicles}
	ctxSrv, cancel := context.WithCancel(context.Background())
	srv := rte.NewRTEServerMockWithRegistry(config.RTEMockConfig{Address: "127.0.0.1:0"}, wrapper, nil, reg)
	go func() { _ = srv.Start(ctxSrv) }()
	time.Sleep(150 * time.Millisecond)
	defer cancel()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	metricsTS := httptest.NewServer(mux)
	defer metricsTS.Close()

	sig := rte.Signal{SignalType: "FCR", StartTime: time.Now(), EndTime: time.Now().Add(2 * time.Minute), Power: 20}
	data, _ := json.Marshal(sig)
	resp, err := http.Post("http://"+srv.Addr()+"/rte/signal", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	time.Sleep(500 * time.Millisecond)

	metricsResp, err := http.Get(metricsTS.URL + "/metrics")
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	body, _ := io.ReadAll(metricsResp.Body)
	if err := metricsResp.Body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}
	out := string(body)
	expectedSignal := `rte_signals_total{signal_type="FCR"} 1`
	if !strings.Contains(out, expectedSignal) {
		t.Errorf("signal metric missing: %s", expectedSignal)
	}
	expected := `dispatch_events_total{acknowledged="true",signal_type="FCR",vehicle_id="veh1"} 1`
	if !strings.Contains(out, expected) {
		t.Errorf("metric missing: %s", expected)
	}
}
