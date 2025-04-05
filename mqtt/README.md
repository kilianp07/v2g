# Documentation for the `mqtt` Package

This package provides an abstraction layer over the MQTT protocol to manage the dispatch of control commands to vehicles and handle acknowledgments (ACKs) received in response. It is built on top of the Eclipse Paho MQTT client.

## Package Overview

### Core Types

#### `DispatchOrder`
Represents a dispatch order sent to a vehicle.
```go
SignalType string  // Type of signal (e.g., "primary", "secondary")
Power      float64 // Power in kW
Duration   int     // Duration in seconds
Timestamp  int64   // UNIX timestamp
```

#### `DispatchAck`
Represents the acknowledgment returned by a vehicle.
```go
VehicleID string // Vehicle identifier
Success   bool   // Indicates success or failure of the order
```

### `FeedbackHandler` Interface
Interface that should be implemented to process incoming ACKs.
```go
type FeedbackHandler interface {
    HandleVehicleFeedback(vehicleID string, success bool)
}
```

### `AckHandler` Function
Creates an MQTT message handler that parses acknowledgments and invokes the provided `FeedbackHandler`.
```go
func AckHandler(fh FeedbackHandler) mqtt.MessageHandler
```

### Topic Utilities
Helper functions to generate MQTT topic names:
```go
func DispatchTopic(vehicleID string) string // Topic for dispatch commands
func AckTopic(vehicleID string) string      // Topic for receiving ACKs
```

## MQTT Client

### `Client` Interface
A minimal abstraction of the MQTT client to facilitate mocking and unit testing.

### `MqttClient` Struct
Encapsulates MQTT connection logic, including publishing and subscribing to topics.

#### Constructor
```go
func NewMqttClient(broker, clientID string, tlsConfig *tls.Config, optsFunc func(*mqtt.ClientOptions)) (*MqttClient, error)
```

#### Methods
```go
Publish(topic string, payload []byte, qos byte) error
Subscribe(topic string, qos byte, cb mqtt.MessageHandler) error
Close() // Gracefully disconnects the client
```

## Quickstart Example

```go
package main

import (
    "crypto/tls"
    "encoding/json"
    "log"
    "time"

    "github.com/kilianp07/v2g/mqtt"
)

type myHandler struct{}

func (h *myHandler) HandleVehicleFeedback(vehicleID string, success bool) {
    log.Printf("ACK received from %s: success=%v", vehicleID, success)
}

func main() {
    client, err := mqtt.NewMqttClient("tls://mqtt-broker:8883", "my-client-id", &tls.Config{}, nil)
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer client.Close()

    vehicleID := "ev-001"

    // Subscribe to ACKs
    if err := client.Subscribe(mqtt.AckTopic(vehicleID), 1, mqtt.AckHandler(&myHandler{})); err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    // Send a dispatch order
    order := mqtt.DispatchOrder{
        SignalType: "primary",
        Power:      5.0,
        Duration:   300,
        Timestamp:  time.Now().Unix(),
    }
    payload, _ := json.Marshal(order)

    if err := client.Publish(mqtt.DispatchTopic(vehicleID), payload, 1); err != nil {
        log.Printf("Failed to publish: %v", err)
    }
}
```

## Unit Tests

Unit tests are included to verify:
- Proper parsing and handling of ACKs via a mock `FeedbackHandler`.
- Correct disconnection behavior via the `Close()` method.

## External Dependencies
- `github.com/eclipse/paho.mqtt.golang`: MQTT library
- `github.com/kilianp07/v2g/logger`: Logging utility for error tracking in the ACK handler

---