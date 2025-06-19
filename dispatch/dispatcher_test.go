package dispatch

import (
	"encoding/json"
	"testing"

	mqttwrapper "github.com/kilianp07/v2g/mqtt"
)

type mockMQTT struct {
	published []string
}

func (m *mockMQTT) Connect(brokerURL string, clientID string, options ...mqttwrapper.ConnectOption) error {
	return nil
}
func (m *mockMQTT) Publish(topic string, payload interface{}, qos byte) error {
	switch v := payload.(type) {
	case []byte:
		m.published = append(m.published, string(v))
	case string:
		m.published = append(m.published, v)
	default:
		b, _ := json.Marshal(v)
		m.published = append(m.published, string(b))
	}
	return nil
}
func (m *mockMQTT) Subscribe(topic string, qos byte, callback mqttwrapper.MessageHandler) error {
	return nil
}
func (m *mockMQTT) Disconnect() {}

func TestDispatch(t *testing.T) {
	vehicles := []Vehicle{{ID: "v1", MaxPower: 5, Available: true, StateOfCharge: 80}}
	m := &mockMQTT{}
	dm, err := NewDispatchManager(vehicles, map[string]float64{"low": 20, "high": 80}, m, "cmd")
	if err != nil {
		t.Fatalf("init error: %v", err)
	}

	sig := FlexibilitySignal{Type: "FCR", Power: 5, Duration: 15}
	dm.Dispatch(sig)

	if len(m.published) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(m.published))
	}
	var order DispatchOrder
	if err := json.Unmarshal([]byte(m.published[0]), &order); err != nil {
		t.Fatalf("invalid payload: %v", err)
	}
	if order.VehicleID != "v1" || order.PowerKW != 5 {
		t.Fatalf("unexpected order %+v", order)
	}
}
