package mqtt

import (
	"fmt"
	"time"
)

// Client represents an MQTT client capable of sending dispatch orders and
// waiting for acknowledgments from vehicles.
type Client interface {
	// SendOrder sends a command to the given vehicle and returns the command
	// identifier used to track the acknowledgment.
	SendOrder(vehicleID string, powerKW float64) (commandID string, err error)

	// WaitForAck waits for an acknowledgment for the provided command
	// identifier or until the timeout expires.
	WaitForAck(commandID string, timeout time.Duration) (bool, error)
}

// MockPublisher is a simple publisher used in tests.
type MockPublisher struct {
	Messages   map[string]float64
	FailIDs    map[string]bool
	AckResults map[string]bool
}

// NewMockPublisher creates a new MockPublisher.
func NewMockPublisher() *MockPublisher {
	return &MockPublisher{
		Messages:   make(map[string]float64),
		FailIDs:    make(map[string]bool),
		AckResults: make(map[string]bool),
	}
}

// SendOrder records the message or returns an error if configured to fail.
func (m *MockPublisher) SendOrder(vehicleID string, powerKW float64) (string, error) {
	if m.FailIDs[vehicleID] {
		return "", fmt.Errorf("publish failed")
	}
	m.Messages[vehicleID] = powerKW
	commandID := fmt.Sprintf("cmd-%s", vehicleID)
	m.AckResults[commandID] = !m.FailIDs[vehicleID]
	return commandID, nil
}

// WaitForAck simulates an immediate acknowledgment based on the stored result.
func (m *MockPublisher) WaitForAck(commandID string, timeout time.Duration) (bool, error) {
	ok, exists := m.AckResults[commandID]
	if !exists {
		return false, fmt.Errorf("unknown command")
	}
	return ok, nil
}
