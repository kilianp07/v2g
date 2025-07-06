package main

import (
	"fmt"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// mqttClientFactory is used to create MQTT clients. It can be swapped out in
// tests.
var mqttClientFactory = realMQTTClient

func newMQTTClient(broker, clientID string) (paho.Client, error) {
	return mqttClientFactory(broker, clientID)
}

func realMQTTClient(broker, clientID string) (paho.Client, error) {
	if broker == "" || clientID == "" {
		return nil, fmt.Errorf("broker and clientID must be provided")
	}
	opts := paho.NewClientOptions().AddBroker(broker).SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectTimeout(10 * time.Second)
	cli := paho.NewClient(opts)
	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	return cli, nil
}
