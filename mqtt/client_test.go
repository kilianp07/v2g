package mqtt

import (
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mockClient struct {
	Disconnected bool
}

func (m *mockClient) IsConnected() bool       { return true }
func (m *mockClient) Disconnect(quiesce uint) { m.Disconnected = true }
func (m *mockClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	return &mockToken{}
}
func (m *mockClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return &mockToken{}
}

type mockToken struct{}

func (t *mockToken) Wait() bool                       { return true }
func (t *mockToken) WaitTimeout(_ time.Duration) bool { return true }
func (t *mockToken) Error() error                     { return nil }
func (t *mockToken) Done() <-chan struct{}            { return make(chan struct{}) }

func TestClose_DisconnectsClient(t *testing.T) {
	mc := &mockClient{}
	client := &MqttClient{client: mc}
	client.Close()
	if !mc.Disconnected {
		t.Errorf("expected Disconnect() to be called")
	}
}
