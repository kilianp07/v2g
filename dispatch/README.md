# Dispatch Package

This package contains the core logic for distributing flexibility signals to electric vehicles. The `DispatchManager` orchestrates filtering vehicles, allocating power and sending orders via an MQTT `Client` implementation.

The manager waits for acknowledgments from each vehicle concurrently. The maximum wait time can be configured through the `ackTimeout` parameter when creating the manager:

```go
manager, err := dispatch.NewDispatchManager(
    dispatch.SimpleVehicleFilter{},
    dispatch.EqualDispatcher{}, // or dispatch.NewSmartDispatcher()
    dispatch.NoopFallback{},
    mqtt.NewMockPublisher(),
    5*time.Second,
    logger.New("dispatch"),
)
```

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
