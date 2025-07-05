package events

import "github.com/kilianp07/v2g/core/model"

// StrategyEvent is emitted when the dispatch manager chooses a dispatcher.
// Action can be "lp_attempt", "lp_failure", or "smart_fallback".
type StrategyEvent struct {
	Signal model.SignalType
	Action string
	Err    error
}
