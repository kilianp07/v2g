package mqttwrapper

import (
	"os"
	"testing"
	"time"

	"encoding/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	m       = message{"test message"}
	mqttUrl = os.Getenv("MQTT_BROKER_URL")
)

// Mock implementations for testing
type MockMQTTClient struct {
	mock.Mock
}

// Test Connect
func TestMQTTClientWrapper_Connect(t *testing.T) {
	client := &MQTTClientWrapper{}
	err := client.Connect(mqttUrl, "go unit test")
	assert.NoError(t, err, "Connect should not return an error")
}

// Test Publish
func TestMQTTClientWrapper_Publish(t *testing.T) {
	client := &MQTTClientWrapper{}

	err := client.Connect(mqttUrl, "go unit test")
	assert.NoError(t, err, "Connect should not return an error")

	err = client.Publish("testtopic", m.ToJson(), 1)
	assert.NoError(t, err, "Publish should not return an error")
}

// Test Subscribe
func TestMQTTClientWrapper_Subscribe(t *testing.T) {
	client := &MQTTClientWrapper{}

	callbackInvoked := false
	callback := func(client MQTTClient, message Message) {
		callbackInvoked = true
		assert.Equal(t, m.ToJson(), string(message.Payload()), "Payload should be equal")
	}

	err := client.Connect(mqttUrl, "go unit test")
	assert.NoError(t, err, "Connect should not return an error")

	err = client.Subscribe("testtopic", 1, callback)
	assert.NoError(t, err, "Subscribe should not return an error")

	err = client.Publish("testtopic", m.ToJson(), 1)
	assert.NoError(t, err, "Subscribe should not return an error")

	// Wait for callback to be invoked
	time.Sleep(1 * time.Second)

	assert.True(t, callbackInvoked, "Callback should be invoked")
}

// Test Disconnect
func TestMQTTClientWrapper_Disconnect(t *testing.T) {
	client := MQTTClientWrapper{}

	err := client.Connect(mqttUrl, "go unit test")
	assert.NoError(t, err, "Connect should not return an error")

	assert.NotPanics(t, func() {
		client.Disconnect()
	}, "Disconnect should not panic")
}

type message struct {
	Message string `json:"message"`
}

func (m *message) ToJson() string {
	text, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(text)
}
