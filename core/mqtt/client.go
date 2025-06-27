package mqtt

import "time"

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
