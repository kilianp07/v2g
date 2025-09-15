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

### Tests E2E Demo-Assurance

The repository ships with a lightweight end-to-end suite exercising a full stack
(Mosquitto, InfluxDB and the V2X service). It is intended to secure the demo
environment by validating that basic telemetry and dispatch flows work.

Run the suite locally with Docker installed:

```bash
make e2e
```

The tests emit a JUnit XML report and coverage file under `coverage/`. A typical
run completes in a couple of minutes.

Integration and end-to-end tests rely on helper utilities under
`test/util`. These helpers provide functions for waiting on the mock RTE
server, launching a Mosquitto broker for MQTT-based tests and polling
Prometheus metrics.

Copy `config.example.yaml` to `config.yaml` and adjust the MQTT credentials to
match your broker.

## RTE Signal Generator (preprod)

An internal generator can emit synthetic RTE flexibility signals for
demonstrations. Enable it in the configuration under `rteGenerator` or via the
CLI flag `--rte-gen`:

```bash
v2g -c config.preprod.yaml --rte-gen --rte-gen-scenario steady
```

Generated signals are published on the internal event bus and recorded through
existing metrics sinks.

## Telemetry

The service can collect vehicle state in two modes:

- **Push** – vehicles periodically publish their state to `state_topic_prefix`.
- **Pull** – the service broadcasts on `request_topic` and awaits responses on `response_topic_prefix`.

Configure the behaviour via the `telemetry` section in the config file or `K_TELEMETRY__*` environment variables.

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

## Error Monitoring with Sentry

Sentry can capture runtime errors and panics during simulations or production.
Enable it by adding a `sentry` section in `config.yaml`:

```yaml
sentry:
  dsn: "https://examplePublicKey@o0.ingest.sentry.io/0"
  environment: "preprod"
  traces_sample_rate: 0.2
  release: "v2g@0.1.0"
```

Leave the `dsn` empty to disable reporting.

## Secure MQTT Client

The `infra/mqtt` package now provides a hardened MQTT client with TLS support,
authentication modes, per-message QoS and automatic retry logic. Last Will
messages can be configured to notify other components of unexpected failures.

Example configuration snippet:

```yaml
mqtt:
  broker: "ssl://broker:8883"
  client_id: "v2g-dispatcher"
  username: "user"
  password: "secret"
  ack_topic: "vehicle/+/ack"
  use_tls: true
  client_cert: "client.crt"
  client_key: "client.key"
  ca_bundle: "ca.pem"
  auth_method: "tls"
  qos:
    command: 1
    ack: 1
  lwt_topic: "v2g/lwt"
  lwt_payload: "offline"
  lwt_qos: 1
  lwt_retain: true
  max_retries: 5
  backoff_ms: 200
```

See `config.example.yaml` for more options.

## Dispatch Logs and API

Every dispatch decision is recorded in a structured log. Logs can be persisted to a SQLite database or JSONL file using the `dispatch.LogStore` implementations. Configure a store and attach it to the manager:

```go
store, _ := dispatch.NewSQLiteStore("dispatch.db")
manager.SetLogStore(store)
```

Logs are exposed through the HTTP handler in `api/dispatch`:

```go
handler := dispatchapi.NewLogHandler(store, "secret-token")
http.Handle("/api/dispatch/logs", handler)
```

Logs rotate automatically when exceeding the configured size or age. Example
configuration:

```yaml
logging:
  backend: "jsonl" # or 'sqlite'
  path: "dispatch.log"
  max_size_mb: 10      # rotate after 10MB
  max_backups: 5       # keep last 5 files
  max_age_days: 7      # purge files older than a week
```

Query records with optional filters:

```
GET /api/dispatch/logs?start=2024-01-02T15:04:05Z&end=2024-01-02T16:00:00Z&vehicle_id=v1&signal_type=FCR
```

Provide the token via `Authorization: Bearer <token>` header.

## Vehicle Status Endpoint

`/api/vehicles/status` exposes the real-time state of each vehicle and last dispatch decision.
Query parameters:

- `fleet_id` – filter by fleet identifier
- `site` – filter by site
- `cluster` – filter by behavioral cluster

Example request:

```bash
GET /api/vehicles/status?fleet_id=f1&site=paris
```

Response sample:

```json
[
  {
    "vehicle_id": "veh123",
    "current_status": "dispatched",
    "forecasted_plugin_window": {
      "start": "2025-07-07T08:00:00Z",
      "end": "2025-07-07T12:00:00Z"
    },
    "forecasted_soc": {
      "t+0m": 80.5,
      "t+15m": 78.2
    },
    "next_dispatch_window": {
      "start": "...",
      "end": "..."
    },
    "last_dispatch_decision": {
      "signal_type": "FCR",
      "target_power": 50.0,
      "vehicles_selected": ["veh123"],
      "timestamp": "2025-07-06T14:30:00Z"
    }
  }
]
```

## Ecological KPIs

The metrics module computes per-vehicle ecological indicators. Configure an emission factor in `config.yaml`:

```yaml
metrics:
  emission_factor: 50 # gCO2 per kWh
```

Prometheus exposes gauges `vehicle_injected_energy_kwh`, `vehicle_co2_avoided_grams`, and `vehicle_energy_ratio` labelled by vehicle and day. The REST endpoint exposes aggregated KPIs:

```bash
GET /api/vehicles/{id}/kpis?start=2025-07-01T00:00:00Z&end=2025-07-07T00:00:00Z
```

Use the backfill job to populate historical data from dispatch logs:

```go
store := eco.NewMemoryStore()
_ = ecokpi.Backfill(store, history)
```


## Day-Ahead Scheduler

The `core/scheduler` package provides a simple day-ahead planning engine used for
NEBEF notifications. Create a `Scheduler` with configuration and a vehicle pool,
then generate a plan for a specific day:

```go
cfg := scheduler.SchedulerConfig{SlotDurationMinutes: 60, TargetEnergyKWh: 24}
s := scheduler.Scheduler{Config: cfg, Vehicles: fleet, Availability: windows}
plan, _ := s.GeneratePlan(time.Now())
```

Plans can be exported as JSON or CSV using the `pkg/export` helpers:

```go
var buf bytes.Buffer
export.WriteJSON(&buf, plan)
export.WriteCSV(&buf, plan)
```

The CSV format uses RTE-compatible headers `vehicle_id,timeslot,power_kw`.
