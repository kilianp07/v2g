package mqtt

import (
	"encoding/json"
	"testing"
)

type fakeHandler struct {
	Called    bool
	VehicleID string
	Success   bool
}

func (f *fakeHandler) HandleVehicleFeedback(vehicleID string, success bool) {
	f.Called = true
	f.VehicleID = vehicleID
	f.Success = success
}

type mockMessage struct {
	payload []byte
}

func (m mockMessage) Duplicate() bool            { return false }
func (m mockMessage) Qos() byte                  { return 1 }
func (m mockMessage) Retained() bool             { return false }
func (m mockMessage) Topic() string              { return "v2g/ack/v123" }
func (m mockMessage) MessageID() uint16          { return 0 }
func (m mockMessage) Payload() []byte            { return m.payload }
func (m mockMessage) Ack()                       {}
func (m mockMessage) Read(b []byte) (int, error) { copy(b, m.payload); return len(m.payload), nil }

func TestAckHandler_ValidAck(t *testing.T) {
	h := &fakeHandler{}
	handler := AckHandler(h)
	ack := DispatchAck{VehicleID: "v123", Success: true}
	payload, _ := json.Marshal(ack)
	msg := mockMessage{payload: payload}

	handler(nil, msg)

	if !h.Called || h.VehicleID != "v123" || !h.Success {
		t.Errorf("unexpected handler result: %+v", h)
	}
}
