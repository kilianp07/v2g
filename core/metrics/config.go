package metrics

import "github.com/kilianp07/v2g/core/factory"

// Config defines settings for metrics sinks.
type Config struct {
	Sinks          []factory.ModuleConfig `json:"sinks"`
	EmissionFactor float64                `json:"emission_factor"`
}
