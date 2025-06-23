# v2g

This repository implements a minimal V2G/V2X dispatch prototype in Go. It distributes flexibility signals to electric vehicles over MQTT and collects acknowledgments.

## Building and Testing

Go 1.20 or later is required.

```bash
# run unit tests
go test ./...
```

Copy `config.example.yaml` to `config.yaml` and adjust the MQTT credentials to
match your broker.

## Packages

- **model**: domain objects such as `Vehicle` and `FlexibilitySignal`.
- **mqtt**: MQTT client interface and implementations.
- **dispatch**: core logic that filters vehicles, allocates power and publishes orders. `SmartDispatcher` provides weighted scoring with fairness and market price awareness. `LPDispatcher` solves a linear program for optimal allocation, and weight tuning can be automated via a `LearningTuner`.
- **logger**: simple logging abstraction with a no-op implementation and a
  Zerolog-based logger for structured output. Use `logger.New(component)` to
  obtain a logger instance. The environment is detected via the `APP_ENV`
  variable.

See individual package READMEs for more details.
