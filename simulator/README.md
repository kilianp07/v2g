# Simulator

This module simulates a fleet of vehicles acknowledging dispatch orders over MQTT.

It can be started with `go run ./simulator` and accepts a few command line
options:

```
--broker        MQTT broker URL
--count         number of vehicles to simulate
--ack-latency   fixed latency before sending the ACK (e.g. `200ms`)
--drop-rate     probability between 0 and 1 to drop an ACK
```

Each simulated vehicle listens on `vehicle/{id}/command` and, according to the
configured strategy, publishes acknowledgments to `vehicle/{id}/ack`.
