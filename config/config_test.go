package config

import (
	"os"
	"path/filepath"
	"testing"
)

//nolint:gocyclo
func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := `mqtt:
  broker: "tcp://localhost:1883"
  client_id: "cli"
  username: "user"
  password: "pass"
  ack_topic: "vehicle/+/ack"
  use_tls: false
dispatch:
  ack_timeout_seconds: 3
  segments:
    commuter:
      dispatcher_type: "heuristic"
    captive_fleet:
      dispatcher_type: "lp"
metrics:
  sinks:
    - type: "nop"
rte:
  mode: "mock"
  mock:
    address: ":9090"
  client:
    poll_interval_seconds: 60
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"broker", cfg.MQTT.Broker, "tcp://localhost:1883"},
		{"client_id", cfg.MQTT.ClientID, "cli"},
		{"username", cfg.MQTT.Username, "user"},
		{"password", cfg.MQTT.Password, "pass"},
		{"ack_topic", cfg.MQTT.AckTopic, "vehicle/+/ack"},
		{"use_tls", cfg.MQTT.UseTLS, false},
		{"ack_timeout_seconds", cfg.Dispatch.AckTimeoutSeconds, 3},
		{"segment", cfg.Dispatch.Segments["captive_fleet"].DispatcherType, "lp"},
		{"metrics_sink", len(cfg.Metrics.Sinks) == 1 && cfg.Metrics.Sinks[0].Type == "nop", true},
		{"rte.mode", cfg.RTE.Mode, "mock"},
		{"rte.mock.address", cfg.RTE.Mock.Address, ":9090"},
		{"rte.client.poll_interval_seconds", cfg.RTE.Client.PollIntervalSeconds, 60},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s mismatch: %v", c.name, c.got)
		}
	}
}
