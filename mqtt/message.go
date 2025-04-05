package mqtt

type DispatchOrder struct {
	SignalType string  `json:"signal_type"`
	Power      float64 `json:"power"`
	Duration   int     `json:"duration"` // en secondes
	Timestamp  int64   `json:"timestamp"`
}

type DispatchAck struct {
	VehicleID string `json:"vehicle_id"`
	Success   bool   `json:"success"`
}
