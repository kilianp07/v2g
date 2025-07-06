# v2g

This repository implements a minimal V2G/V2X dispatch prototype in Go. It
distributes flexibility signals to electric vehicles over MQTT and collects
acknowledgments.

## Building and Testing

Go 1.20 or later is required.

```bash
# run unit tests
go test ./...
```

Copy `config.example.yaml` to `config.yaml` and adjust the MQTT credentials to
match your broker.

## Package Layout

- The code is progressively migrating towards a layered architecture:

- **core/** – pure business logic such as dispatch algorithms, domain models
  and event definitions.
- **infra/** – technical adapters (MQTT clients, metrics exporters, etc.).
- **app/** – orchestration layer wiring the service together.
- **cmd/** – CLI entry points invoking the application service.

The legacy packages have been migrated:
`core/dispatch`, `core/model`, `core/metrics` host the business
types and algorithms while infrastructure implementations live in
`infra/mqtt`, `infra/metrics`, and `infra/logger`.
Event definitions live under `core/events`.

See individual package READMEs for details.

## Prediction Engine

`DispatchManager` can optionally use a `PredictionEngine` to forecast vehicle availability and state of charge. Provide an implementation when creating the manager to improve scoring:

```go
pred := &prediction.MockPredictionEngine{}
mgr, _ := dispatch.NewDispatchManager(
    dispatch.SimpleVehicleFilter{},
    dispatch.EqualDispatcher{},
    dispatch.NoopFallback{},
    mqtt.NewMockPublisher(),
    5*time.Second,
    metrics.NopSink{},
    eventbus.New(),
    nil,
    logger.NopLogger{},
    nil,
    pred,
)
```

The `MockPredictionEngine` returns deterministic values and is used in tests. Custom engines can be plugged in the same way.

## Metrics

Prometheus metrics are registered automatically when importing the `dispatch` package. Start the HTTP server to expose them:

```go
ctx := context.Background()
go metrics.StartPromServer(ctx, ":2112")
```

Key metrics:
- `dispatch_execution_latency_seconds` – histogram of publish-to-ack latency per signal type
- `vehicles_dispatched_total` – counter of vehicles dispatched per signal type
- `ack_rate` – gauge representing acknowledged ratio per dispatch
- `mqtt_publish_success_total` / `mqtt_publish_failure_total` – MQTT publish results

Configure your Prometheus scrape job to target the `/metrics` endpoint.
