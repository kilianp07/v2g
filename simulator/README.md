# Simulator

This module simulates a fleet of vehicles acknowledging dispatch orders over MQTT.

It can be started with `go run ./simulator` and accepts several command line
options:

```
--broker           MQTT broker URL
--count            number of vehicles to simulate
--fleet-size       auto-generate N vehicles
--commuter-pct     ratio of commuter vehicles (0-1)
--disconnect-rate  per-minute disconnect probability
--availability-file hourly availability profile JSON
--schedule-file    optional schedule overrides
--template-file    vehicle template overrides
--ack-latency   fixed latency before sending the ACK (e.g. `200ms`)
--drop-rate     probability between 0 and 1 to drop an ACK
--capacity      battery capacity in kWh
--charge-rate   maximum charge rate in kW
--discharge-rate maximum discharge rate in kW
--max-power     vehicle power limit in kW
--interval      publish interval for SoC metrics
--topic-prefix  MQTT topic prefix (default "v2g")
--battery-profile battery size preset (small, medium, large)
--verbose       enable verbose logging
--influx-url    InfluxDB URL
--influx-token  InfluxDB token
--influx-org    InfluxDB organization
--influx-bucket InfluxDB bucket
```

Each simulated vehicle subscribes to `vehicle/{id}/command` and, according to the
configured strategy, publishes acknowledgments to `vehicle/{id}/ack`. It
periodically publishes its SoC on `<prefix>/vehicle/state/{id}` and answers the
`<prefix>/fleet/discovery` broadcast by sending a status message to
`<prefix>/fleet/response/{id}`.
