// Package dispatch implements the core logic for orchestrating flexibility
// dispatch in a V2X (Vehicle-to-Everything) energy system.
//
// It receives real-time flexibility signals from the grid operator (e.g. FCR, aFRR, MA,
// NEBEF, EcoWatt) and distributes the requested power — injection or reduction — across
// a fleet of electric vehicles (EVs), respecting energy, mobility, and market constraints.
//
// # Dispatch Flow
//
//  1. Filter eligible vehicles.
//  2. Allocate power using a dispatcher strategy.
//  3. Publish commands via MQTT (mockable).
//  4. Collect acknowledgments or failures.
//  5. Trigger fallback if necessary.
//
// # Components
//
// - DispatchManager: Central coordinator for filtering, dispatching, publishing, fallback.
// - VehicleFilter: Selects eligible EVs (e.g., based on SoC, V2G capability).
// - Dispatcher:
//   - EqualDispatcher: Evenly distributes power respecting MaxPower.
//   - SmartDispatcher: Greedy scoring with weights (SoC, priority, time, price, fairness).
//   - LPDispatcher: Linear programming optimization using Gonum.
//
// - FallbackStrategy: Handles reallocation in case of failed dispatch.
// - LearningTuner: (Optional) Tunes SmartDispatcher weights based on past results.
//
// # SmartDispatcher Scoring
//
// Each vehicle receives a score based on:
//   - Energy slack (SoC - MinSoC)
//   - Time to planned departure
//   - Charging priority flag
//   - Market price sensitivity (e.g., EcoWatt)
//   - Participation history (fairness)
//   - Signal type (affects weight balancing)
//
// Power is allocated proportionally to scores, over multiple rounds
// (bounded by MaxRounds) to meet the requested power.
//
// # Example
//
//	disp := dispatch.NewSmartDispatcher()
//	disp.Participation["veh42"] = 5 // penalize overused vehicle
//	disp.MaxRounds = 3
//	assignments := disp.Dispatch(vehicles, signal)
//
// # Guarantees
//
// - All dispatchers enforce physical constraints (SoC, MaxPower, availability).
// - LPDispatcher ensures optimal allocation when needed.
// - Components are decoupled via interfaces and easily testable.
// - Thread-safe, production-ready, and aligned with market-based flexibility requirements.
//
// This package is the heart of V2X orchestration logic, designed to support
// reliable, real-time dispatch in response to grid needs.
package dispatch
