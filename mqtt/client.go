package mqtt

import (
	"crypto/tls"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client interface {
	IsConnected() bool
	Disconnect(uint)
	Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token
	Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token
}

type MqttClient struct {
	client Client
}

func NewMqttClient(broker, clientID string, tlsConfig *tls.Config, optsFunc func(*mqtt.ClientOptions)) (*MqttClient, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetTLSConfig(tlsConfig).
		SetConnectTimeout(5 * time.Second).
		SetAutoReconnect(true)

	if optsFunc != nil {
		optsFunc(opts)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	return &MqttClient{client: client}, nil
}

func (mc *MqttClient) Publish(topic string, payload []byte, qos byte) error {
	token := mc.client.Publish(topic, qos, false, payload)
	token.Wait()
	return token.Error()
}

func (mc *MqttClient) Subscribe(topic string, qos byte, cb mqtt.MessageHandler) error {
	token := mc.client.Subscribe(topic, qos, cb)
	token.Wait()
	return token.Error()
}

func (mc *MqttClient) Close() {
	if mc.client.IsConnected() {
		mc.client.Disconnect(250)
	}
}
