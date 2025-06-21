# Dispatch Package

This package contains the core logic for distributing flexibility signals to electric vehicles. The `DispatchManager` orchestrates filtering vehicles, allocating power and sending orders via an MQTT `Client` implementation.

The manager waits for acknowledgments from each vehicle concurrently. The maximum wait time can be configured through the `ackTimeout` parameter when creating the manager:

```go
manager, err := dispatch.NewDispatchManager(
    dispatch.SimpleVehicleFilter{},
    dispatch.EqualDispatcher{},
    dispatch.NoopFallback{},
    mqtt.NewMockPublisher(),
    5*time.Second,
)
```

If an order fails or no ACK is received before the timeout, a fallback strategy can reallocate the remaining power.
