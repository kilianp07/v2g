package dispatch

import (
	"math"

	"github.com/kilianp07/v2g/logger"
	"github.com/sirupsen/logrus"
)

// HandleVehicleFeedback processes confirmations from vehicles after dispatch.
func (dm *DispatchManager) HandleVehicleFeedback(vehicleID string, success bool) {
	if !success {
		logger.Log.WithFields(logrus.Fields{
			"vehicle_id": vehicleID,
		}).Warn("Vehicle failed to execute dispatch order. Reallocating power.")
		dm.ReallocatePower(vehicleID)
	}
}

// ReallocatePower attempts to redistribute power to other available vehicles.
func (dm *DispatchManager) ReallocatePower(failedVehicleID string) float64 {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	logger.Log.WithFields(logrus.Fields{
		"failed_vehicle_id": failedVehicleID,
	}).Info("Attempting power reallocation.")

	// Mark the failed vehicle as unavailable and get its power
	remainingPower := 0.0
	found := false
	for i := range dm.vehicles {
		if dm.vehicles[i].ID == failedVehicleID {
			dm.vehicles[i].Available = false
			remainingPower = dm.vehicles[i].MaxPower
			found = true
			break
		}
	}
	if !found {
		logger.Log.WithField("vehicle_id", failedVehicleID).Warn("Failed vehicle not found in vehicle list.")
		return 0
	}

	initialPower := remainingPower
	if remainingPower > 0 {
		logger.Log.WithFields(logrus.Fields{
			"failed_vehicle_id":   failedVehicleID,
			"power_to_reallocate": remainingPower,
		}).Info("Reallocating power to available vehicles.")

		for i := range dm.vehicles {
			if dm.vehicles[i].ID == failedVehicleID {
				continue
			}
			thresholdLow := dm.configurableSOC["low"]
			if dm.vehicles[i].Available && dm.vehicles[i].StateOfCharge > thresholdLow {
				allocatedPower := math.Min(remainingPower, dm.vehicles[i].MaxPower)
				// Keep the vehicle available for future dispatches

				logger.Log.WithFields(logrus.Fields{
					"vehicle_id":      dm.vehicles[i].ID,
					"allocated_power": allocatedPower,
				}).Info("Reallocated power to vehicle.")

				remainingPower -= allocatedPower
				if remainingPower <= 0 {
					break
				}
			}
		}
	} else {
		logger.Log.Warn("No remaining power to reallocate.")
	}

	return initialPower - remainingPower
}
