package config

import (
	"os"
	"path/filepath"
	"testing"
)

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
  prometheus_enabled: false
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
	if cfg.MQTT.Broker != "tcp://localhost:1883" {
		t.Errorf("broker mismatch: %s", cfg.MQTT.Broker)
	}
	if cfg.MQTT.ClientID != "cli" {
		t.Errorf("client_id mismatch: %s", cfg.MQTT.ClientID)
	}
	if cfg.MQTT.Username != "user" {
		t.Errorf("username mismatch: %s", cfg.MQTT.Username)
	}
	if cfg.MQTT.Password != "pass" {
		t.Errorf("password mismatch: %s", cfg.MQTT.Password)
	}
	if cfg.MQTT.AckTopic != "vehicle/+/ack" {
		t.Errorf("ack_topic mismatch: %s", cfg.MQTT.AckTopic)
	}
	if cfg.MQTT.UseTLS {
		t.Errorf("use_tls mismatch: expected false")
	}
	if cfg.Dispatch.AckTimeoutSeconds != 3 {
		t.Errorf("ack_timeout_seconds mismatch: %d", cfg.Dispatch.AckTimeoutSeconds)
	}
	if cfg.Dispatch.Segments["captive_fleet"].DispatcherType != "lp" {
		t.Errorf("segment not parsed")
	}
	if cfg.RTE.Mode != "mock" {
		t.Errorf("rte.mode mismatch: %s", cfg.RTE.Mode)
	}
	if cfg.RTE.Mock.Address != ":9090" {
		t.Errorf("rte.mock.address mismatch: %s", cfg.RTE.Mock.Address)
	}
	if cfg.RTE.Client.PollIntervalSeconds != 60 {
		t.Errorf("rte.client.poll_interval_seconds mismatch: %d", cfg.RTE.Client.PollIntervalSeconds)
	}
}
