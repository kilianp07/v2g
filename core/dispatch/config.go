package dispatch

import "github.com/kilianp07/v2g/core/model"

// Config defines dispatch-related settings.
type Config struct {
	AckTimeoutSeconds int                       `json:"ack_timeout_seconds"`
	LPFirst           map[model.SignalType]bool `json:"lp_first"`
}
