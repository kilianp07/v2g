# Metrics Schema

The following measurements are exported to InfluxDB with a 7‑day retention policy.

## vehicle_soc_percent
*Tags*: `vehicle_id`, `fleet_id`
*Fields*:
- `soc` (float) – State of charge in percent

## dispatch_order_kw
*Tags*: `vehicle_id`, `signal_type`, `mode`
*Fields*:
- `power_kw` (float)

## acknowledgment
*Tags*: `vehicle_id`, `signal_type`
*Fields*:
- `acknowledged` (bool)
- `latency_ms` (float)

## signal_metadata
*Tags*: `signal_type`
*Fields*:
- `power_kw` (float)

## vehicle_availability
*Tags*: `vehicle_id`
*Fields*:
- `probability` (float)

All timestamps correspond to the event time.
