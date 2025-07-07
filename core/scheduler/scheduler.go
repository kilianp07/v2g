package scheduler

import (
	"errors"
	"fmt"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

// AvailabilityWindow represents a plug-in period for a vehicle.
type AvailabilityWindow struct {
	Start time.Time
	End   time.Time
}

// SchedulerConfig defines planning parameters loaded from configuration.
type SchedulerConfig struct {
	SlotDurationMinutes int     `json:"slot_duration_minutes" yaml:"slot_duration_minutes"`
	TargetEnergyKWh     float64 `json:"target_energy_kwh" yaml:"target_energy_kwh"`
}

// Scheduler generates day-ahead effacement plans.
type Scheduler struct {
	Config       SchedulerConfig
	Vehicles     []model.Vehicle
	Availability map[string][]AvailabilityWindow
}

// GeneratePlan builds an effacement plan for the given day.
// It returns one entry per vehicle and timeslot.
func (s *Scheduler) GeneratePlan(date time.Time) ([]EffacementEntry, error) {
	if s.Config.SlotDurationMinutes <= 0 {
		return nil, errors.New("slot_duration_minutes must be positive")
	}
	slotDur := time.Duration(s.Config.SlotDurationMinutes) * time.Minute
	totalSlots := int((24 * time.Hour) / slotDur)
	if totalSlots == 0 {
		return nil, errors.New("slot duration too long")
	}

	powerPerSlot := s.Config.TargetEnergyKWh / (float64(totalSlots) * slotDur.Hours())
	if powerPerSlot <= 0 {
		return nil, errors.New("target energy must be positive")
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	var entries []EffacementEntry

	for i := 0; i < totalSlots; i++ {
		ts := startOfDay.Add(time.Duration(i) * slotDur)
		var available []model.Vehicle
		for _, v := range s.Vehicles {
			if s.vehicleAvailable(v.ID, ts, slotDur) {
				available = append(available, v)
			}
		}
		if len(available) == 0 {
			return nil, fmt.Errorf("no vehicles available at %v", ts)
		}
		totalCap := 0.0
		for _, v := range available {
			totalCap += v.MaxPower
		}
		if totalCap < powerPerSlot {
			return nil, fmt.Errorf("insufficient capacity at %v", ts)
		}
		share := powerPerSlot / float64(len(available))
		for _, v := range available {
			p := share
			if p > v.MaxPower {
				p = v.MaxPower
			}
			entries = append(entries, EffacementEntry{
				VehicleID: v.ID,
				TimeSlot:  ts,
				PowerKW:   p,
			})
		}
	}
	return entries, nil
}

func (s *Scheduler) vehicleAvailable(id string, t time.Time, d time.Duration) bool {
	windows := s.Availability[id]
	end := t.Add(d)
	for _, w := range windows {
		if (t.Equal(w.Start) || t.After(w.Start)) && !end.After(w.End) {
			return true
		}
	}
	return false
}

// EffacementEntry defines a vehicle allocation for a specific timeslot.
type EffacementEntry struct {
	VehicleID string    `json:"vehicle_id"`
	TimeSlot  time.Time `json:"timeslot"`
	PowerKW   float64   `json:"power_kw"`
}
