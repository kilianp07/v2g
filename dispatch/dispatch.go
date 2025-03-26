package dispatch

import (
	"github.com/kilianp07/v2g/logger"

	"github.com/sirupsen/logrus"
)

// ExecuteDispatch processes selected vehicles and logs actions.
func (dm *DispatchManager) ExecuteDispatch(signal FlexibilitySignal) {
	selectedVehicles, powerAllocation, err := dm.SelectVehicles(signal)
	if err != nil {
		logger.Log.Error("Failed to select vehicles for dispatch: %w", err)
		return
	}

	if len(selectedVehicles) == 0 {
		logger.Log.Warn("No vehicles selected for this signal.")
		return
	}

	for _, v := range selectedVehicles {
		logger.Log.WithFields(logrus.Fields{
			"vehicle_id":      v.ID,
			"priority":        v.Priority,
			"soc":             v.StateOfCharge,
			"tariff":          v.Tariff,
			"signal_type":     signal.Type,
			"power_allocated": powerAllocation[v.ID],
		}).Info("Vehicle responding to dispatch order")
	}
}

// MonitorExecution tracks and logs dispatch performance.
func MonitorExecution(signal FlexibilitySignal, selectedVehicles []Vehicle, powerAllocation map[string]float64) {
	logger.Log.WithFields(logrus.Fields{
		"type":     signal.Type,
		"power":    signal.Power,
		"duration": signal.Duration,
	}).Info("Execution summary")
}
