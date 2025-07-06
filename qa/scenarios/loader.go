package scenarios

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kilianp07/v2g/core/model"
)

type VehicleDef struct {
	ID         string  `yaml:"id"`
	SoC        float64 `yaml:"soc"`
	IsV2G      bool    `yaml:"is_v2g"`
	Available  bool    `yaml:"available"`
	MaxPower   float64 `yaml:"max_power"`
	BatteryKWh float64 `yaml:"battery_kwh"`
}

func (v VehicleDef) ToModel() model.Vehicle {
	return model.Vehicle{
		ID:         v.ID,
		SoC:        v.SoC,
		IsV2G:      v.IsV2G,
		Available:  v.Available,
		MaxPower:   v.MaxPower,
		BatteryKWh: v.BatteryKWh,
	}
}

type SignalDef struct {
	Type            string  `yaml:"type"`
	PowerKW         float64 `yaml:"power_kw"`
	DurationSeconds int     `yaml:"duration_seconds"`
}

func (s SignalDef) ToModel() model.FlexibilitySignal {
	return model.FlexibilitySignal{
		Type:      parseSignalType(s.Type),
		PowerKW:   s.PowerKW,
		Duration:  time.Duration(s.DurationSeconds) * time.Second,
		Timestamp: time.Now(),
	}
}

type Expected struct {
	Acked int `yaml:"acked"`
}

type Scenario struct {
	Name            string         `yaml:"name"`
	Description     string         `yaml:"description,omitempty"`
	Vehicles        []VehicleDef   `yaml:"vehicles"`
	Signals         []SignalDef    `yaml:"signals"`
	FailVehicles    []string       `yaml:"fail_vehicles,omitempty"`
	AckFailAfter    map[string]int `yaml:"ack_fail_after,omitempty"`
	DisconnectAfter map[string]int `yaml:"disconnect_after,omitempty"`
	Expected        Expected       `yaml:"expected"`
}

func Load(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sc Scenario
	if err := yaml.Unmarshal(data, &sc); err != nil {
		return nil, err
	}
	return &sc, nil
}

func parseSignalType(t string) model.SignalType {
	switch t {
	case "FCR":
		return model.SignalFCR
	case "aFRR":
		return model.SignalAFRR
	case "MA":
		return model.SignalMA
	case "NEBEF":
		return model.SignalNEBEF
	case "EcoWatt":
		return model.SignalEcoWatt
	default:
		return model.SignalFCR
	}
}
