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
)
```

Passing a custom tuner allows the dispatcher weights to adapt automatically
after each dispatch based on historical results.

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

The `Vehicle` model exposes an `EffectiveCapacity(current)` helper that computes
the usable power capacity of a vehicle based on its SoC, estimated availability
and degradation. Both fallback strategies rely on this method to ensure
consistent capacity estimates.

### References
- Liu et al. (2020) – *A Reliability-Aware Vehicle-to-Grid Scheduling Strategy in Smart Grid*
- Deng et al. (2021) – *Probabilistic Load Dispatch Considering EV Uncertainty*
- Zhao et al. (2019) – *SoC-Constrained Dispatch in Aggregated V2G Fleets*
