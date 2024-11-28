package mqttwrapper

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTClient defines the methods for an MQTT client wrapper.
type MQTTClient interface {
	Connect(brokerURL string, clientID string, options ...ConnectOption) error
	Publish(topic string, payload interface{}, qos byte) error
	Subscribe(topic string, qos byte, callback MessageHandler) error
	Disconnect()
}

// MessageHandler is a wrapper around mqtt.MessageHandler for easier abstraction.
type MessageHandler func(client MQTTClient, message Message)

// Message wraps mqtt.Message for further abstraction.
type Message interface {
	Topic() string
	Payload() []byte
	Duplicate() bool
	QoS() byte
	Retained() bool
}

// mqttMessage is an internal implementation of the Message interface.
type mqttMessage struct {
	msg mqtt.Message
}

func (m *mqttMessage) Topic() string   { return m.msg.Topic() }
func (m *mqttMessage) Payload() []byte { return m.msg.Payload() }
func (m *mqttMessage) Duplicate() bool { return m.msg.Duplicate() }
func (m *mqttMessage) QoS() byte       { return m.msg.Qos() }
func (m *mqttMessage) Retained() bool  { return m.msg.Retained() }

// ConnectOption defines a functional option for configuring the MQTT client.
type ConnectOption func(opts *mqtt.ClientOptions)

// WithPasswordAuth adds username and password authentication to the MQTT client.
func WithPasswordAuth(username, password string) ConnectOption {
	return func(opts *mqtt.ClientOptions) {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}
}

// MQTTClientWrapper implements MQTTClient using paho.mqtt.golang.
type MQTTClientWrapper struct {
	client mqtt.Client
}

// Connect establishes a connection to the MQTT broker with optional configurations.
func (m *MQTTClientWrapper) Connect(brokerURL string, clientID string, options ...ConnectOption) error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)

	// Apply additional options
	for _, option := range options {
		option(opts)
	}

	m.client = mqtt.NewClient(opts)
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to broker: %v", token.Error())
	}
	return nil
}

// Publish sends a message to the specified MQTT topic.
func (m *MQTTClientWrapper) Publish(topic string, payload interface{}, qos byte) error {
	if token := m.client.Publish(topic, qos, false, payload); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish message: %v", token.Error())
	}
	return nil
}

// Subscribe subscribes to a topic with a callback function.
func (m *MQTTClientWrapper) Subscribe(topic string, qos byte, callback MessageHandler) error {
	wrappedCallback := func(client mqtt.Client, msg mqtt.Message) {
		callback(m, &mqttMessage{msg: msg})
	}
	if token := m.client.Subscribe(topic, qos, wrappedCallback); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic: %v", token.Error())
	}
	return nil
}

// Disconnect gracefully disconnects from the MQTT broker.
func (m *MQTTClientWrapper) Disconnect() {
	m.client.Disconnect(250) // Gracefully disconnect with a timeout
}
