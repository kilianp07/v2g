package dispatch

import (
	"encoding/json"
	"fmt"
	"testing"

	mqttwrapper "github.com/kilianp07/v2g/mqtt"
)

type mockMQTT struct {
	published  []string
	publishErr error
}

func (m *mockMQTT) Connect(brokerURL string, clientID string, options ...mqttwrapper.ConnectOption) error {
	return nil
}
func (m *mockMQTT) Publish(topic string, payload interface{}, qos byte) error {
	if m.publishErr != nil {
		return m.publishErr
	}
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
	tests := []struct {
		name              string
		vehicles          []Vehicle
		signal            FlexibilitySignal
		publishErr        error
		expectedPublishes int
	}{
		{
			name:              "single vehicle",
			vehicles:          []Vehicle{{ID: "v1", MaxPower: 5, Available: true, StateOfCharge: 80}},
			signal:            FlexibilitySignal{Type: "FCR", Power: 5, Duration: 15},
			expectedPublishes: 1,
		},
		{
			name: "multiple vehicles partial availability",
			vehicles: []Vehicle{
				{ID: "v1", MaxPower: 5, Available: false, StateOfCharge: 90},
				{ID: "v2", MaxPower: 10, Available: true, StateOfCharge: 50},
			},
			signal:            FlexibilitySignal{Type: "FCR", Power: 4, Duration: 10},
			expectedPublishes: 1,
		},
		{
			name:              "insufficient total power",
			vehicles:          []Vehicle{{ID: "v1", MaxPower: 3, Available: true, StateOfCharge: 90}},
			signal:            FlexibilitySignal{Type: "FCR", Power: 10, Duration: 5},
			expectedPublishes: 1,
		},
		{
			name:              "zero power",
			vehicles:          []Vehicle{{ID: "v1", MaxPower: 5, Available: true, StateOfCharge: 90}},
			signal:            FlexibilitySignal{Type: "FCR", Power: 0, Duration: 10},
			expectedPublishes: 1,
		},
		{
			name:              "no vehicles",
			vehicles:          nil,
			signal:            FlexibilitySignal{Type: "FCR", Power: 5, Duration: 10},
			expectedPublishes: 0,
		},
		{
			name:              "publish error",
			vehicles:          []Vehicle{{ID: "v1", MaxPower: 5, Available: true, StateOfCharge: 90}},
			signal:            FlexibilitySignal{Type: "FCR", Power: 5, Duration: 10},
			publishErr:        fmt.Errorf("failure"),
			expectedPublishes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockMQTT{publishErr: tt.publishErr}
			dm, err := NewDispatchManager(tt.vehicles, map[string]float64{"low": 20, "high": 80}, m, "cmd")
			if err != nil {
				t.Fatalf("init error: %v", err)
			}

			dm.Dispatch(tt.signal)

			if len(m.published) != tt.expectedPublishes {
				t.Fatalf("expected %d publish, got %d", tt.expectedPublishes, len(m.published))
			}
			if tt.expectedPublishes > 0 && tt.signal.Power == 0 {
				var order DispatchOrder
				if err := json.Unmarshal([]byte(m.published[0]), &order); err != nil {
					t.Fatalf("invalid payload: %v", err)
				}
				if order.PowerKW != 0 {
					t.Fatalf("expected power 0 got %.2f", order.PowerKW)
				}
			}
		})
	}
}
