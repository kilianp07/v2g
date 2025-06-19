package mqttwrapper

import (
	"os"
	"testing"

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
	if mqttUrl == "" {
		t.Skip("MQTT_BROKER_URL not set")
	}
	client := &MQTTClientWrapper{}
	err := client.Connect(mqttUrl, "go unit test", WithPasswordAuth(os.Getenv("MQTT_USERNAME"), os.Getenv("MQTT_PASSWORD")))
	assert.NoError(t, err, "Connect should not return an error")
}

// Test Publish
func TestMQTTClientWrapper_Publish(t *testing.T) {
	if mqttUrl == "" {
		t.Skip("MQTT_BROKER_URL not set")
	}
	client := &MQTTClientWrapper{}

	err := client.Connect(mqttUrl, "go unit test", WithPasswordAuth(os.Getenv("MQTT_USERNAME"), os.Getenv("MQTT_PASSWORD")))
	assert.NoError(t, err, "Connect should not return an error")

	err = client.Publish("testtopic", m.ToJson(), 1)
	assert.NoError(t, err, "Publish should not return an error")
}

// Test Disconnect
func TestMQTTClientWrapper_Disconnect(t *testing.T) {
	if mqttUrl == "" {
		t.Skip("MQTT_BROKER_URL not set")
	}
	client := MQTTClientWrapper{}

	err := client.Connect(mqttUrl, "go unit test", WithPasswordAuth(os.Getenv("MQTT_USERNAME"), os.Getenv("MQTT_PASSWORD")))
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
