# MQTT Package

This package provides an interface for sending power commands to vehicles via
MQTT and waiting for acknowledgment messages. Commands are published on
`vehicle/{vehicle_id}/command` and acknowledgments are expected on a shared
`vehicle/+/ack` topic. Messages include a `command_id` (UUID) and timestamp.
It contains a mock implementation used in tests as well as a production client
based on the Eclipse Paho library with automatic reconnection and optional TLS
support. Logging is performed via the `logger` package which defines an
interface covering debug, info, warning and error levels, along with a
`NopLogger` implementation used by default. A `ZerologLogger` based on
[`rs/zerolog`](https://github.com/rs/zerolog) is available for structured
logging.
