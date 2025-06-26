package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// SimulatedVehicle connects to MQTT and acknowledges commands.
type SimulatedVehicle struct {
	ID       string
	Broker   string
	Strategy AckStrategy

	client paho.Client
	ackCh  chan string
}

// NewSimulatedVehicle creates a new vehicle.
func NewSimulatedVehicle(id, broker string, strat AckStrategy) *SimulatedVehicle {
	return &SimulatedVehicle{
		ID:       id,
		Broker:   broker,
		Strategy: strat,
		ackCh:    make(chan string, 50),
	}
}

// Run connects to the broker and listens for commands until ctx is done.
func (v *SimulatedVehicle) Run(ctx context.Context) error {
	cli, err := newMQTTClient(v.Broker, "sim-"+v.ID)
	if err != nil {
		return err
	}
	v.client = cli
	for i := 0; i < 5; i++ {
		go v.worker(ctx)
	}
	topic := fmt.Sprintf("vehicle/%s/command", v.ID)
	if token := cli.Subscribe(topic, 0, v.onCommand(ctx)); token.Wait() && token.Error() != nil {
		cli.Disconnect(250)
		return token.Error()
	}
	<-ctx.Done()
	close(v.ackCh)
	cli.Disconnect(250)
	return nil
}

func (v *SimulatedVehicle) onCommand(ctx context.Context) func(paho.Client, paho.Message) {
	return func(_ paho.Client, msg paho.Message) {
		var m struct {
			CommandID string `json:"command_id"`
		}
		if err := json.Unmarshal(msg.Payload(), &m); err != nil {
			log.Printf("%s: decode command: %v", v.ID, err)
			return
		}
		select {
		case v.ackCh <- m.CommandID:
		default:
			log.Printf("%s: ack queue full, dropping command %s", v.ID, m.CommandID)
		}
	}
}

func (v *SimulatedVehicle) worker(ctx context.Context) {
	for {
		select {
		case cmdID, ok := <-v.ackCh:
			if !ok {
				return
			}
			v.Strategy.Ack(ctx, v.client, v.ID, cmdID)
		case <-ctx.Done():
			return
		}
	}
}
