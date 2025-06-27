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
