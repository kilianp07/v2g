# Dispatch Package Documentation

## Overview

The `dispatch` package is responsible for managing the selection, dispatching, and monitoring of electric vehicles (EVs) in response to flexibility signals issued by a grid operator. It implements logic to select the most appropriate vehicles, handle dispatch failures, and ensure reliable energy reallocation in line with system requirements.

---

## Types

### `FlexibilitySignal`

Represents a request from the grid for flexibility in energy dispatch.

```go
type FlexibilitySignal struct {
    Type        string    // Type of signal: "PRIMARY_RESERVE", "SECONDARY_RESERVE", or "LOAD_SHEDDING"
    Power       float64   // Requested power adjustment in kW
    Duration    int       // Duration of the flexibility event in minutes
    Timestamp   time.Time // Timestamp when the signal was issued
    MarketPrice float64   // Current market price in €/kWh
}
```

---

### `Vehicle`

Models an electric vehicle that may participate in V2G (Vehicle-to-Grid) operations.

```go
type Vehicle struct {
    ID            string    // Unique identifier of the vehicle
    StateOfCharge float64   // Battery charge level as a percentage (0-100)
    MaxPower      float64   // Maximum power capacity in kW
    Available     bool      // Availability status for dispatch
    Priority      int       // Priority score for dispatch selection
    Tariff        float64   // Current applicable tariff in €/kWh
    LastUpdate    time.Time // Last update time of vehicle status
}
```

---

### `DispatchManager`

Coordinates vehicle dispatching and system logic.

```go
type DispatchManager struct {
    vehicles        []Vehicle              // List of managed vehicles
    mutex           sync.RWMutex           // Thread-safe locking
    configurableSOC map[string]float64     // SOC thresholds: e.g., {"low": 20.0, "high": 80.0}
}
```

---

## Constructors

### `NewDispatchManager`

Initializes a new `DispatchManager` instance with SOC validation.

```go
func NewDispatchManager(vehicles []Vehicle, socThresholds map[string]float64) (*DispatchManager, error)
```

- Returns an error if thresholds are invalid.
- Ensures `low < high` and all thresholds are within 0–100%.

---

## Core Methods

### `(*DispatchManager) SelectVehicles`

Determines the best vehicles to respond to a given flexibility signal based on priority, tariff, and SOC constraints.

```go
func (dm *DispatchManager) SelectVehicles(signal FlexibilitySignal) ([]Vehicle, map[string]float64, error)
```

- Prioritizes vehicles based on:
  - Higher priority score
  - Higher SOC (when required)
  - Lower tariffs
- Considers SOC thresholds depending on signal type.

---

### `(*DispatchManager) ExecuteDispatch`

Executes dispatch logic:

- Selects vehicles for the provided signal
- Logs the dispatched vehicles and parameters
- Handles selection errors or empty results

```go
func (dm *DispatchManager) ExecuteDispatch(signal FlexibilitySignal)
```

---

### `MonitorExecution`

Logs the summary of a dispatch execution.

```go
func MonitorExecution(signal FlexibilitySignal, selectedVehicles []Vehicle, powerAllocation map[string]float64)
```

---

### `(*DispatchManager) UpdateVehicleState`

Updates the status of a specific vehicle, including its SOC and availability.

```go
func (dm *DispatchManager) UpdateVehicleState(vehicleID string, soc float64, available bool)
```

---

### `(*DispatchManager) HandleVehicleFeedback`

Handles the response feedback from a vehicle post-dispatch. If dispatch failed, triggers a reallocation of the power.

```go
func (dm *DispatchManager) HandleVehicleFeedback(vehicleID string, success bool)
```

---

### `(*DispatchManager) ReallocatePower`

Reallocates power from a failed vehicle to other eligible and available vehicles.

```go
func (dm *DispatchManager) ReallocatePower(failedVehicleID string)
```

- Skips vehicles with SOC below 20%
- Allocates as much power as possible up to `MaxPower`
- Stops reallocating once the original power is fulfilled

---

## Testing

The test suite validates:

- Dispatch and reallocation logic
- Correct vehicle prioritization
- Behavior when no vehicles are available
- Error handling and logging

Test utilities include:

- `fakeDispatcher`: A stub for `SelectVehicles`
- Custom error injection via `assertError`

All tests are implemented using Go’s `testing` package.

---

## Example

```go
dm, err := NewDispatchManager(vehicles, map[string]float64{"low": 20.0, "high": 80.0})
if err != nil {
    log.Fatal(err)
}

signal := FlexibilitySignal{
    Type:     "PRIMARY_RESERVE",
    Power:    10,
    Duration: 15,
}

dm.ExecuteDispatch(signal)
```