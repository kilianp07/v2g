package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

var fleetRng = rand.New(rand.NewSource(time.Now().UnixNano()))

// FleetConfig holds parameters for bulk fleet generation.
type FleetConfig struct {
	Size           int
	CommuterPct    float64
	DisconnectRate float64
	Availability   [24]float64
	Schedule       map[string]time.Time
}

// VehicleTemplate allows overriding generated vehicles.
type VehicleTemplate struct {
	Departure string `json:"departure"`
}

// GenerateFleet creates Size vehicles with IDs veh0001..vehNNNN.
// Vehicles are assigned the "commuter" segment according to CommuterPct,
// otherwise "opportunistic".
func GenerateFleet(cfg FleetConfig, tmpl map[string]VehicleTemplate) []SimulatedVehicle {
	if cfg.Size <= 0 {
		return nil
	}
	vs := make([]SimulatedVehicle, cfg.Size)
	for i := 0; i < cfg.Size; i++ {
		id := fmt.Sprintf("veh%04d", i+1)
		seg := "opportunistic"
		if cfg.CommuterPct > 0 && fleetRng.Float64() < cfg.CommuterPct {
			seg = "commuter"
		}
		dep := time.Time{}
		if t, ok := cfg.Schedule[id]; ok {
			dep = t
		} else if tmpl != nil {
			if v, ok := tmpl[id]; ok {
				if v.Departure != "" {
					if tt, err := time.Parse(time.RFC3339, v.Departure); err == nil {
						dep = tt
					}
				}
			}
		}
		vs[i] = SimulatedVehicle{
			ID:             id,
			Segment:        seg,
			DisconnectRate: cfg.DisconnectRate,
			Availability:   cfg.Availability,
			Departure:      dep,
		}
	}
	return vs
}

// LoadAvailabilityProfile reads a hourly availability profile from JSON or YAML.
func LoadAvailabilityProfile(data []byte) ([24]float64, error) {
	var m map[string]float64
	var prof [24]float64
	if err := json.Unmarshal(data, &m); err != nil {
		return prof, err
	}
	for h, v := range m {
		var hour int
		if _, err := fmt.Sscanf(h, "%d", &hour); err != nil {
			continue
		}
		if hour >= 0 && hour < 24 {
			prof[hour] = v
		}
	}
	return prof, nil
}
