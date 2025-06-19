package dispatch

import "time"

// DispatchOrder represents a command sent to a vehicle over MQTT.
type DispatchOrder struct {
	VehicleID string    `json:"vehicle_id"`
	PowerKW   float64   `json:"power_kw"`
	Duration  int       `json:"duration_min"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}
