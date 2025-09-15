# Metrics

The service writes the following InfluxDB measurements:

- **vehicle_state**
  - tags: `vehicle_id`
  - fields: `soc` (float), `status` (string), `power_kw` (float)
  - example: `vehicle_state,vehicle_id=veh1 soc=0.57,status="charging",power_kw=3.2`
- **dispatch_order**
  - tags: `vehicle_id`, `signal_type`, `order_id`
  - fields: `power_kw` (float), `score` (float), `accepted` (bool)
- **acknowledgment**
  - tags: `vehicle_id`, `order_id`
  - fields: `ack` (bool), `latency_ms` (float)
- **signal**
  - tags: `signal_type`
  - fields: `power_requested_kw` (float), `duration_s` (int)

Run the application with `--demo-seed` to write sample points and verify
connectivity.
