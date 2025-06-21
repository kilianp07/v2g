package mqtt

import "fmt"

// Publisher represents an MQTT publisher used to send power commands to vehicles.
type Publisher interface {
	Publish(vehicleID string, powerKW float64) error
}

// MockPublisher is a simple publisher used in tests.
type MockPublisher struct {
	Messages map[string]float64
	FailIDs  map[string]bool
}

// NewMockPublisher creates a new MockPublisher.
func NewMockPublisher() *MockPublisher {
	return &MockPublisher{Messages: make(map[string]float64), FailIDs: make(map[string]bool)}
}

// Publish records the message or returns an error if configured to fail.
func (m *MockPublisher) Publish(vehicleID string, powerKW float64) error {
	if m.FailIDs[vehicleID] {
		return fmt.Errorf("publish failed")
	}
	m.Messages[vehicleID] = powerKW
	return nil
}
