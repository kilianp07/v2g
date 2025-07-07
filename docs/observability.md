# Observability Guide

This document explains how to configure and interpret the dispatch logs.

## Log Levels

Set the `APP_ENV` environment variable to `dev` to enable human readable console
logs. The default JSON format is recommended for production.

## Enabling Structured Logs

Logging is configured through the `logging` section of `config.yaml`:

```yaml
logging:
  backend: "jsonl" # or 'sqlite'
  path: "dispatch.log"
  max_size_mb: 10
  max_backups: 5
  max_age_days: 7
```

* `backend` – selects the log storage driver.
* `path` – file path for the log.
* `max_size_mb` – rotate when the file exceeds this size.
* `max_backups` – number of rotated files to retain.
* `max_age_days` – remove backups older than this many days.

## Log Fields

Each dispatch decision is stored with the following fields:

| Field | Description |
|-------|-------------|
| `timestamp` | time of the dispatch |
| `signal` | flexibility signal details |
| `target_power` | requested power in kW |
| `vehicles_selected` | list of vehicles participating |
| `response` | assignments, scores and acknowledgment results |

Debug logs include structured JSON objects describing scores, fallback triggers
and timing information useful for troubleshooting.

## Troubleshooting

1. Increase the log level to `debug` using the `APP_ENV=dev` environment
   variable when running the service.
2. Inspect the most recent log file in the location configured above. Rotated
   files have timestamps appended to the name.
3. Use `api/dispatch/logs` to query records for a specific vehicle or time
   window.
