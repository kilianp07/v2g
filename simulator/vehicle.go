package main

import (
	"context"
	"encoding/json"
	"fmt"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// SimulatedVehicle connects to MQTT and acknowledges commands.
type SimulatedVehicle struct {
	ID       string
	Broker   string
	Strategy AckStrategy

	client paho.Client
}

// NewSimulatedVehicle creates a new vehicle.
func NewSimulatedVehicle(id, broker string, strat AckStrategy) *SimulatedVehicle {
	return &SimulatedVehicle{ID: id, Broker: broker, Strategy: strat}
}

// Run connects to the broker and listens for commands until ctx is done.
func (v *SimulatedVehicle) Run(ctx context.Context) error {
	cli, err := newMQTTClient(v.Broker, "sim-"+v.ID)
	if err != nil {
		return err
	}
	v.client = cli
	topic := fmt.Sprintf("vehicle/%s/command", v.ID)
	if token := cli.Subscribe(topic, 0, v.onCommand(ctx)); token.Wait() && token.Error() != nil {
		cli.Disconnect(250)
		return token.Error()
	}
	<-ctx.Done()
	cli.Disconnect(250)
	return nil
}

func (v *SimulatedVehicle) onCommand(ctx context.Context) func(paho.Client, paho.Message) {
	return func(_ paho.Client, msg paho.Message) {
		var m struct {
			CommandID string `json:"command_id"`
		}
		if err := json.Unmarshal(msg.Payload(), &m); err != nil {
			return
		}
		go v.Strategy.Ack(ctx, v.client, v.ID, m.CommandID)
	}
}
