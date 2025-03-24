package dispatch

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/kilianp07/v2g/logger"
	"github.com/sirupsen/logrus"
)

// DispatchManager manages V2G dispatch decisions.
type DispatchManager struct {
	vehicles        []Vehicle
	mutex           sync.RWMutex
	configurableSOC map[string]float64 // Configurable SOC thresholds
}

// NewDispatchManager initializes a new DispatchManager.
func NewDispatchManager(vehicles []Vehicle, socThresholds map[string]float64) (*DispatchManager, error) {
	if socThresholds["low"] < 0 || socThresholds["low"] > 100 || socThresholds["high"] < 0 || socThresholds["high"] > 100 {
		return nil, errors.New("SOC thresholds must be between 0 and 100")
	}
	if socThresholds["low"] >= socThresholds["high"] {
		return nil, errors.New("'low' threshold must be less than 'high' threshold")
	}

	return &DispatchManager{
		vehicles:        vehicles,
		configurableSOC: socThresholds,
	}, nil
}

// UpdateVehicleState dynamically updates the state of vehicles.
func (dm *DispatchManager) UpdateVehicleState(vehicleID string, soc float64, available bool) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	for i := range dm.vehicles {
		if dm.vehicles[i].ID == vehicleID {
			dm.vehicles[i].StateOfCharge = soc
			dm.vehicles[i].Available = available
			dm.vehicles[i].LastUpdate = time.Now()
			logger.Log.WithFields(logrus.Fields{
				"vehicle_id":      vehicleID,
				"state_of_charge": soc,
				"available":       available,
			}).Info("Vehicle state updated")
			break
		}
	}
}

// SelectVehicles determines the optimal vehicles to respond to a flexibility signal.
func (dm *DispatchManager) SelectVehicles(signal FlexibilitySignal) ([]Vehicle, map[string]float64, error) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	if len(dm.vehicles) == 0 {
		return nil, nil, nil
	}

	selected := []Vehicle{}
	powerAllocation := make(map[string]float64)
	remainingPower := signal.Power

	sort.Slice(dm.vehicles, func(i, j int) bool {
		if dm.vehicles[i].Priority == dm.vehicles[j].Priority {
			if dm.vehicles[i].StateOfCharge == dm.vehicles[j].StateOfCharge {
				return dm.vehicles[i].Tariff < dm.vehicles[j].Tariff
			}
			return dm.vehicles[i].StateOfCharge > dm.vehicles[j].StateOfCharge
		}
		return dm.vehicles[i].Priority > dm.vehicles[j].Priority
	})

	thresholdLow := dm.configurableSOC["low"]
	thresholdHigh := dm.configurableSOC["high"]

	for i := range dm.vehicles {
		if !dm.vehicles[i].Available {
			continue
		}
		allocatedPower := math.Min(remainingPower, dm.vehicles[i].MaxPower)
		if (signal.Type == "LOAD_SHEDDING" && dm.vehicles[i].StateOfCharge < thresholdHigh) ||
			(signal.Type != "LOAD_SHEDDING" && dm.vehicles[i].StateOfCharge > thresholdLow) {
			selected = append(selected, dm.vehicles[i])
			powerAllocation[dm.vehicles[i].ID] = allocatedPower
			remainingPower -= allocatedPower
			if remainingPower <= 0 {
				break
			}
		}
	}

	if len(selected) == 0 {
		return nil, nil, nil
	}

	return selected, powerAllocation, nil
}
