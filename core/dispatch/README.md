# Dispatch Package

This package contains the core logic for distributing flexibility signals to electric vehicles. The `DispatchManager` orchestrates filtering vehicles, allocating power and sending orders via an MQTT `Client` implementation. It can optionally discover the fleet dynamically via a `FleetDiscovery` component.

The manager waits for acknowledgments from each vehicle concurrently. The maximum wait time can be configured through the `ackTimeout` parameter when creating the manager:

```go
manager, err := dispatch.NewDispatchManager(
    dispatch.SimpleVehicleFilter{},
    dispatch.EqualDispatcher{}, // or dispatch.NewSmartDispatcher()
    dispatch.NoopFallback{},
    mqtt.NewMockPublisher(),
    5*time.Second,
    metrics.NopSink{},
    eventbus.New(),
    nil, // FleetDiscovery
    logger.NopLogger{},
    nil, // LearningTuner
    nil, // PredictionEngine
)
```

Passing a custom tuner allows the dispatcher weights to adapt automatically
after each dispatch based on historical results.

### Dynamic Weight Tuning

`AckBasedTuner` provides a simple feedback loop that increases or decreases the
`AvailabilityWeight` of a `SmartDispatcher` depending on acknowledgment rates.
You can create it with defaults:

```go
tuner := dispatch.NewAckBasedTuner(&dispatcher)
```

Or use `NewAckBasedTunerWithConfig` to provide custom steps and thresholds. A
`nil` return indicates an invalid configuration.

`SmartDispatcher` exposes weighting factors that can be tuned per signal type.
Its features are normalized so weights are easier to interpret. You can also
track a participation score per vehicle to ensure fairness across dispatches.
For optimal allocation, `LPDispatcher` provides a linear-programming variant
that maximizes the weighted score. Both dispatchers can be tuned dynamically by
implementing the `LearningTuner` interface. Power allocation loops are bounded
by the `MaxRounds` field:

```go
disp := dispatch.NewSmartDispatcher()
disp.Participation["veh42"] = 5 // penalise overused vehicle
disp.MaxRounds = 5
```

If an order fails or no ACK is received before the timeout, a fallback strategy can reallocate the remaining power.
`BalancedFallback` redistributes the residual power among the vehicles that acknowledged, using their remaining capacity weighted by their state of charge. Vehicles below 30% SoC are skipped.

| Fallback | Description |
|----------|-------------|
| `NoopFallback` | Leaves assignments unchanged |
| `BalancedFallback` | Redistributes residual power proportionally to remaining capacity and SoC |
| `ProbabilisticFallback` | Uses availability probabilities and degradation factors to reallocate power |

### LP-First Dispatch

`DispatchManager` can prioritize the `LPDispatcher` for services that require strict power compliance, such as FCR. Configure the behaviour with the `lp_first` map:

```yaml
dispatch:
  ack_timeout_seconds: 5
  lp_first:
    "0": true # FCR
```

When enabled for a signal type, the manager attempts an LP-based allocation first and falls back to `SmartDispatcher` if the solver fails or is infeasible.

### Segmented Smart Dispatcher

`SegmentedSmartDispatcher` applies distinct scoring weights and dispatch strategies per vehicle segment. Each `Vehicle` can specify a `Segment` label. Configure segments in `dispatch.segments`:

```yaml
dispatch:
  segments:
    commuter:
      dispatcher_type: "heuristic"
      weights:
        soc: 0.6
        time: 0.3
        priority: 0.1
    captive_fleet:
      dispatcher_type: "lp"
      fallback: true
      weights:
        soc: 0.4
        time: 0.2
        availability: 0.4
    opportunistic_charger:
      dispatcher_type: "heuristic"
      fallback: true
      weights:
        soc: 0.2
        time: 0.1
        price: 0.4
        availability: 0.2
        wear: 0.1
```

Missing or unknown segments revert to default `SmartDispatcher` weights.

The `Vehicle` model exposes an `EffectiveCapacity(current)` helper that computes
the usable power capacity of a vehicle based on its SoC, estimated availability
and degradation. Both fallback strategies rely on this method to ensure
consistent capacity estimates.

### References
- Liu et al. (2020) – *A Reliability-Aware Vehicle-to-Grid Scheduling Strategy in Smart Grid*
- Deng et al. (2021) – *Probabilistic Load Dispatch Considering EV Uncertainty*
- Zhao et al. (2019) – *SoC-Constrained Dispatch in Aggregated V2G Fleets*
