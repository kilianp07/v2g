package dispatch

import (
	"testing"
	"time"
)

func TestMonitorExecution(t *testing.T) {
	signal := FlexibilitySignal{
		Type:     "primary",
		Power:    10.0,
		Duration: int(time.Minute * 10),
	}
	vehicles := []Vehicle{
		{ID: "v1"},
		{ID: "v2"},
	}
	allocation := map[string]float64{
		"v1": 6.0,
		"v2": 4.0,
	}

	// Just run it to verify no panic and log output is generated
	MonitorExecution(signal, vehicles, allocation)
}

type fakeDispatcher struct {
	DispatchManager
	selectVehiclesFunc func(FlexibilitySignal) ([]Vehicle, map[string]float64, error)
}

func (f *fakeDispatcher) SelectVehicles(signal FlexibilitySignal) ([]Vehicle, map[string]float64, error) {
	return f.selectVehiclesFunc(signal)
}

func TestExecuteDispatch(t *testing.T) {
	tests := []struct {
		name                string
		selectVehiclesFunc  func(FlexibilitySignal) ([]Vehicle, map[string]float64, error)
		expectVehicleLogged bool
	}{
		{
			name: "error selecting vehicles",
			selectVehiclesFunc: func(FlexibilitySignal) ([]Vehicle, map[string]float64, error) {
				return nil, nil, assertError("selection failed")
			},
			expectVehicleLogged: false,
		},
		{
			name: "no vehicles selected",
			selectVehiclesFunc: func(FlexibilitySignal) ([]Vehicle, map[string]float64, error) {
				return []Vehicle{}, nil, nil
			},
			expectVehicleLogged: false,
		},
		{
			name: "vehicles selected",
			selectVehiclesFunc: func(FlexibilitySignal) ([]Vehicle, map[string]float64, error) {
				return []Vehicle{
						{ID: "v1", Priority: 1, StateOfCharge: 80.0, Tariff: 0.15},
					}, map[string]float64{
						"v1": 5.0,
					}, nil
			},
			expectVehicleLogged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := &fakeDispatcher{selectVehiclesFunc: tt.selectVehiclesFunc}
			dm.ExecuteDispatch(FlexibilitySignal{Type: "secondary"})
			// No assertion: relies on log visibility; validate no panic
		})
	}
}

func assertError(msg string) error {
	return &testError{msg}
}

type testError struct {
	s string
}

func (e *testError) Error() string {
	return e.s
}
