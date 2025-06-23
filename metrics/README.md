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
