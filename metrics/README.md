# Metrics Package

This package provides a `MetricsSink` interface to record dispatch results. The
`InfluxSink` implementation writes events to InfluxDB using the official client.
Each instance can be created with `NewInfluxSinkWithFallback`, which pings the
database and returns a `NopSink` if the health check fails. Each vehicle
allocation generates one `dispatch_event` measurement with the following schema:

- **Tags**: `vehicle_id`, `signal_type`, `acknowledged`
- **Fields**: `power_kw`, `score`, `market_price`, `dispatch_time`
  (the point timestamp equals the signal timestamp and `dispatch_time` records
  when the event was persisted)

Example query to retrieve the history of a vehicle:

```flux
from(bucket: "v2g")
  |> range(start: -1d)
  |> filter(fn: (r) => r._measurement == "dispatch_event" and r.vehicle_id == "veh42")
```

To get all events for a given signal type:

```flux
from(bucket: "v2g")
  |> range(start: -1d)
  |> filter(fn: (r) => r._measurement == "dispatch_event" and r.signal_type == "FCR")
```

`PromSink` offers a Prometheus alternative. Use `NewPromSink` to register both a
`dispatch_events_total` counter and a `dispatch_latency_seconds` histogram on a
given registry (the default registry is used when `nil` is provided). Each
dispatch result increments the counter with labels `vehicle_id`, `signal_type`
and `acknowledged`. Latency between command send and acknowledgment is observed
with the same labels.

When using multiple sinks (e.g. Prometheus and InfluxDB) you can combine them
with `NewMultiSink`:

```go
prom := metrics.NewPromSink(nil)
influx := metrics.NewInfluxSink("http://influx:8086", "token", "org", "bucket", nil)
sink := metrics.NewMultiSink(prom, influx)
```

Expose the metrics with:

```go
go metrics.StartPromServer(":2112")
```
