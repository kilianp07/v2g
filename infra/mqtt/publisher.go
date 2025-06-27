package mqtt

import (
	"fmt"
	"sync"
	"time"

	coremqtt "github.com/kilianp07/v2g/core/mqtt"
)

// Client mirrors the core mqtt.Client interface.
type Client = coremqtt.Client

// MockPublisher is a simple publisher used in tests.
type MockPublisher struct {
	Messages   map[string]float64
	FailIDs    map[string]bool
	AckResults map[string]bool
	mu         sync.Mutex
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
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.Lock()
	ok, exists := m.AckResults[commandID]
	m.mu.Unlock()
	if !exists {
		return false, fmt.Errorf("unknown command")
	}
	return ok, nil
}
