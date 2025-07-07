package scheduler

import (
	"errors"
	"fmt"
	"sort"
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

func (s *Scheduler) availableVehicles(ts time.Time, d time.Duration) []model.Vehicle {
	var res []model.Vehicle
	for _, v := range s.Vehicles {
		if s.vehicleAvailable(v.ID, ts, d) {
			res = append(res, v)
		}
	}
	return res
}

func distribute(ts time.Time, vehicles []model.Vehicle, target float64) []EffacementEntry {
	sort.Slice(vehicles, func(i, j int) bool { return vehicles[i].MaxPower > vehicles[j].MaxPower })
	remaining := target
	alloc := make([]float64, len(vehicles))
	for remaining > 1e-6 {
		active := 0
		for i, v := range vehicles {
			if alloc[i] < v.MaxPower {
				active++
			}
		}
		if active == 0 {
			break
		}
		share := remaining / float64(active)
		progress := false
		for i, v := range vehicles {
			if alloc[i] >= v.MaxPower {
				continue
			}
			give := share
			if alloc[i]+give > v.MaxPower {
				give = v.MaxPower - alloc[i]
			}
			if give <= 0 {
				continue
			}
			alloc[i] += give
			remaining -= give
			progress = true
			if remaining <= 1e-6 {
				break
			}
		}
		if !progress {
			break
		}
	}
	var entries []EffacementEntry
	for i, v := range vehicles {
		if alloc[i] > 0 {
			entries = append(entries, EffacementEntry{VehicleID: v.ID, TimeSlot: ts, PowerKW: alloc[i]})
		}
	}
	return entries
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

	if len(s.Vehicles) == 0 {
		return nil, errors.New("no vehicles configured")
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	var entries []EffacementEntry
	backlog := 0.0 // kWh
	hadAvail := false

	for i := 0; i < totalSlots; i++ {
		ts := startOfDay.Add(time.Duration(i) * slotDur)
		available := s.availableVehicles(ts, slotDur)
		slotEnergy := powerPerSlot * slotDur.Hours()
		if len(available) == 0 {
			backlog += slotEnergy
			continue
		}

		hadAvail = true
		totalCap := 0.0
		for _, v := range available {
			totalCap += v.MaxPower
		}
		targetEnergy := slotEnergy + backlog
		maxEnergy := totalCap * slotDur.Hours()
		allocEnergy := targetEnergy
		if allocEnergy > maxEnergy {
			allocEnergy = maxEnergy
		}
		backlog = targetEnergy - allocEnergy
		targetPower := allocEnergy / slotDur.Hours()
		entries = append(entries, distribute(ts, available, targetPower)...)
	}

	if backlog > 1e-6 || !hadAvail {
		return nil, fmt.Errorf("insufficient capacity for target energy")
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
