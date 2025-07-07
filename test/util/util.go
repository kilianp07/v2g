// Package util provides helper functions shared across integration tests.
//
// WaitForRTEServer polls the mock RTE server until its /rte/ping endpoint
// becomes available.
//
// StartMosquitto launches a disposable Mosquitto broker in a Docker container
// for MQTT-based tests. It returns the broker URL and a cleanup function.
//
// WaitForMetric polls a Prometheus metrics endpoint until the desired metric
// appears in the output.
package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kilianp07/v2g/core/dispatch"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/rte"
)

const (
	// Default timeouts for helper operations
	RTEServerTimeout      = 5 * time.Second
	MosquittoReadyTimeout = 5 * time.Second
	MetricTimeout         = 5 * time.Second

	pollInterval = 50 * time.Millisecond
)

// ManagerWrapper implements the rte.Dispatcher interface and forwards
// dispatches to the provided manager with a fixed set of vehicles.
// It is useful for tests that expose a DispatchManager through the RTE mock
// server without implementing the full API client.
type ManagerWrapper struct {
	Mgr      *dispatch.DispatchManager
	Vehicles []model.Vehicle
}

// Dispatch proxies the call to the underlying manager with the configured
// vehicles.
func (m ManagerWrapper) Dispatch(sig model.FlexibilitySignal, _ []model.Vehicle) dispatch.DispatchResult {
	return m.Mgr.Dispatch(sig, m.Vehicles)
}

// WaitForRTEServer polls the mock RTE server health endpoint until it responds
// with HTTP 200 or the context is done.
func WaitForRTEServer(ctx context.Context, srv *rte.RTEServerMock) error {
	for {
		url := "http://" + srv.Addr() + "/rte/ping"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("server not ready: %w", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// WaitForMetric polls the given metrics URL until the provided substring is
// found in the output or the context is done.
func WaitForMetric(ctx context.Context, metricsURL, substr string) error {
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, metricsURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			body, rerr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if rerr != nil {
				return fmt.Errorf("read metrics body: %w", rerr)
			}
			if strings.Contains(string(body), substr) {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("metric %q not found: %w", substr, ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// StartMosquitto launches a temporary Mosquitto broker inside a Docker
// container and returns its broker URL along with a cleanup function.
func StartMosquitto(ctx context.Context) (string, func(), error) {
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

	dir, err := os.MkdirTemp("", "mosq")
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(dir, "mosquitto.conf")
	if err := os.WriteFile(path, []byte(conf), 0644); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
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
		_ = os.RemoveAll(dir)
		return "", nil, err
	}

	cleanup := func() {
		_ = cont.Terminate(context.Background())
		_ = os.RemoveAll(dir)
	}

	host, err := cont.Host(ctx)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	port, err := cont.MappedPort(ctx, "1883")
	if err != nil {
		cleanup()
		return "", nil, err
	}
	broker := fmt.Sprintf("tcp://%s:%s", host, port.Port())

	waitCtx, cancel := context.WithTimeout(ctx, MosquittoReadyTimeout)
	defer cancel()
	if err := waitForMQTTReady(waitCtx, broker); err != nil {
		cleanup()
		return "", nil, err
	}

	return broker, cleanup, nil
}

func waitForMQTTReady(ctx context.Context, broker string) error {
	opts := paho.NewClientOptions().AddBroker(broker).SetClientID("probe")
	for {
		cli := paho.NewClient(opts)
		token := cli.Connect()
		token.Wait()
		if token.Error() == nil {
			cli.Disconnect(100)
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
