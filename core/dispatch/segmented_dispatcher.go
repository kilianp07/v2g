package dispatch

import (
	"github.com/kilianp07/v2g/core/model"
)

// SegmentedSmartDispatcher applies different dispatcher configurations per vehicle segment.
type SegmentedSmartDispatcher struct {
	segments map[string]SegmentConfig
}

// SegmentConfig defines weights and strategy for a segment.
type SegmentConfig struct {
	Weights        map[string]float64 `json:"weights"`
	DispatcherType string             `json:"dispatcher_type"`
	Fallback       bool               `json:"fallback"`
}

// DefaultSegmentConfigs returns rule-based defaults for common fleet segments.
func DefaultSegmentConfigs() map[string]SegmentConfig {
	return map[string]SegmentConfig{
		"commuter": {
			Weights: map[string]float64{
				"soc":      0.6,
				"time":     0.3,
				"priority": 0.1,
			},
			DispatcherType: "heuristic",
		},
		"captive_fleet": {
			Weights: map[string]float64{
				"soc":          0.4,
				"time":         0.2,
				"availability": 0.4,
			},
			DispatcherType: "lp",
			Fallback:       true,
		},
		"opportunistic_charger": {
			Weights: map[string]float64{
				"soc":          0.2,
				"time":         0.1,
				"price":        0.4,
				"availability": 0.2,
				"wear":         0.1,
			},
			DispatcherType: "heuristic",
			Fallback:       true,
		},
	}
}

// NewSegmentedSmartDispatcher creates a dispatcher with the provided segment configurations.
// If cfg is nil, DefaultSegmentConfigs are used.
func NewSegmentedSmartDispatcher(cfg map[string]SegmentConfig) *SegmentedSmartDispatcher {
	if cfg == nil {
		cfg = DefaultSegmentConfigs()
	}
	cp := make(map[string]SegmentConfig, len(cfg))
	for k, v := range cfg {
		cp[k] = v
	}
	return &SegmentedSmartDispatcher{segments: cp}
}

func applyWeights(d *SmartDispatcher, weights map[string]float64) {
	for k, v := range weights {
		switch k {
		case "soc":
			d.SocWeight = v
		case "time":
			d.TimeWeight = v
		case "priority":
			d.PriorityWeight = v
		case "price":
			d.PriceWeight = v
		case "wear":
			d.WearWeight = v
		case "fairness":
			d.FairnessWeight = v
		case "availability":
			d.AvailabilityWeight = v
		case "market_price":
			d.MarketPrice = v
		}
	}
}

func (d *SegmentedSmartDispatcher) dispatchSegment(vs []model.Vehicle, signal model.FlexibilitySignal, cfg SegmentConfig) map[string]float64 {
	sd := NewSmartDispatcher()
	applyWeights(&sd, cfg.Weights)

	if cfg.DispatcherType == "lp" {
		lp := NewLPDispatcher()
		lp.SmartDispatcher = sd
		if cfg.Fallback {
			asn, err := lp.DispatchStrict(vs, signal)
			if err != nil {
				return sd.Dispatch(vs, signal)
			}
			return asn
		}
		return lp.Dispatch(vs, signal)
	}
	return sd.Dispatch(vs, signal)
}

// Dispatch allocates power per segment using the configured strategy for each.
func (d *SegmentedSmartDispatcher) Dispatch(vehicles []model.Vehicle, signal model.FlexibilitySignal) map[string]float64 {
	assignments := make(map[string]float64)
	if len(vehicles) == 0 {
		return assignments
	}
	groups := make(map[string][]model.Vehicle)
	for _, v := range vehicles {
		groups[v.Segment] = append(groups[v.Segment], v)
	}
	share := signal.PowerKW / float64(len(groups))
	for seg, vs := range groups {
		cfg, ok := d.segments[seg]
		if !ok {
			cfg = SegmentConfig{}
		}
		part := signal
		part.PowerKW = share
		res := d.dispatchSegment(vs, part, cfg)
		for id, p := range res {
			assignments[id] = p
		}
	}
	return assignments
}
