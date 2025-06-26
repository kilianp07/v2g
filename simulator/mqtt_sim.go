package main

import paho "github.com/eclipse/paho.mqtt.golang"

func newMQTTClient(broker, clientID string) (paho.Client, error) {
	opts := paho.NewClientOptions().AddBroker(broker).SetClientID(clientID)
	opts.AutoReconnect = true
	cli := paho.NewClient(opts)
	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	return cli, nil
}
