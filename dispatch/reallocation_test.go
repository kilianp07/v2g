package dispatch

import (
	"testing"
)

func TestReallocatePower(t *testing.T) {
	tests := []struct {
		name             string
		vehicles         []Vehicle
		failedVehicleID  string
		expectedStates   map[string]bool
		expectedPowerOut float64
	}{
		{
			name: "vehicle not found",
			vehicles: []Vehicle{
				{ID: "v1", Available: true, MaxPower: 10, StateOfCharge: 50},
			},
			failedVehicleID: "vX",
			expectedStates: map[string]bool{
				"v1": true,
			},
			expectedPowerOut: 0,
		},
		{
			name: "no remaining power to reallocate",
			vehicles: []Vehicle{
				{ID: "v1", Available: true, MaxPower: 0, StateOfCharge: 50},
				{ID: "v2", Available: true, MaxPower: 10, StateOfCharge: 50},
			},
			failedVehicleID: "v1",
			expectedStates: map[string]bool{
				"v1": false,
				"v2": true,
			},
			expectedPowerOut: 0,
		},
		{
			name: "full reallocation to one vehicle",
			vehicles: []Vehicle{
				{ID: "v1", Available: true, MaxPower: 5, StateOfCharge: 50},
				{ID: "v2", Available: true, MaxPower: 10, StateOfCharge: 50},
			},
			failedVehicleID: "v1",
			expectedStates: map[string]bool{
				"v1": false,
				"v2": false,
			},
			expectedPowerOut: 0,
		},
		{
			name: "partial reallocation due to SOC",
			vehicles: []Vehicle{
				{ID: "v1", Available: true, MaxPower: 5, StateOfCharge: 50},
				{ID: "v2", Available: true, MaxPower: 10, StateOfCharge: 10}, // low SOC
				{ID: "v3", Available: true, MaxPower: 2, StateOfCharge: 50},
			},
			failedVehicleID: "v1",
			expectedStates: map[string]bool{
				"v1": false,
				"v2": true,
				"v3": false,
			},
			expectedPowerOut: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := &DispatchManager{
				vehicles: tt.vehicles,
			}
			dm.ReallocatePower(tt.failedVehicleID)

			for _, v := range dm.vehicles {
				if want, got := tt.expectedStates[v.ID], v.Available; want != got {
					t.Errorf("vehicle %s available state: want %v, got %v", v.ID, want, got)
				}
			}
		})
	}
}
