// Package dispatch implements the core logic for dispatching flexibility
// signals in a V2X (Vehicle-to-Everything) energy system.
//
// It receives real-time flexibility signals (e.g. FCR, aFRR, MA, NEBEF, EcoWatt)
// and distributes the requested power (injection or reduction) among a fleet
// of electric vehicles (EVs) based on their availability, technical constraints,
// and mobility requirements.
//
// Key components:
//   - DispatchManager: orchestrates filtering, allocation, publishing, and fallback.
//   - VehicleFilter: filters EVs according to the type of signal.
//   - Dispatcher: distributes power among selected vehicles.
//   - FallbackStrategy: handles reallocation if dispatch to some vehicles fails.
//
// Default implementations include:
//   - SimpleVehicleFilter: filters based on signal type, SoC, V2G capability.
//   - EqualDispatcher: evenly splits power between vehicles, respecting MaxPower.
//   - NoopFallback: no reallocation on failure.
//
// Dispatch flow:
//  1. Filter vehicles
//  2. Dispatch power
//  3. Publish commands via MQTT (mockable interface)
//  4. Collect acknowledgments or errors
//  5. Trigger fallback strategy if needed
//
// All components are decoupled via interfaces, supporting testing and extension.
// Tests are located in dispatch_test.go and cover filtering, dispatching,
// manager logic and fallback behavior.
//
// Usage example:
//
//	manager, err := dispatch.NewDispatchManager(
//	        dispatch.SimpleVehicleFilter{},
//	        dispatch.EqualDispatcher{},
//	        dispatch.NoopFallback{},
//	        mqtt.NewMockPublisher(),
//	        5*time.Second,
//	)
//	if err != nil {
//	        log.Fatalf("failed to create manager: %v", err)
//	}
//	result := manager.Dispatch(signal, vehicles)
//
// This package follows SOLID principles, is thread-safe, and production-ready.
package dispatch
