# v2g

This repository implements a minimal V2G/V2X dispatch prototype in Go. It distributes flexibility signals to electric vehicles over MQTT and collects acknowledgments.

## Building and Testing

Go 1.20 or later is required.

```bash
# run unit tests
go test ./...
```

## Packages

- **model**: domain objects such as `Vehicle` and `FlexibilitySignal`.
- **mqtt**: MQTT client interface and implementations.
- **dispatch**: core logic that filters vehicles, allocates power and publishes orders.
- **logger**: simple logging abstraction.

See individual package READMEs for more details.
