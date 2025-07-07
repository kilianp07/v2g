package dispatch

import "github.com/kilianp07/v2g/core/model"

// Config defines dispatch-related settings.
type Config struct {
	AckTimeoutSeconds    int                       `json:"ack_timeout_seconds"`
	LPFirst              map[model.SignalType]bool `json:"lp_first"`
	Segments             map[string]SegmentConfig  `json:"segments"`
	EnableSoCConstraints bool                      `json:"enable_soc_constraints"`
	MinSoC               float64                   `json:"min_soc"`
	SafeDischargeFloor   float64                   `json:"safe_discharge_floor"`
}
