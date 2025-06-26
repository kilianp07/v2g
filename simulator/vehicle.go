package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/kilianp07/v2g/model"
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
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v.worker(ctx)
		}()
	}
	if token := cli.Subscribe("v2g/fleet/discovery", 0, v.onDiscovery()); token.Wait() && token.Error() != nil {
		cli.Disconnect(250)
		return token.Error()
	}

	topic := fmt.Sprintf("vehicle/%s/command", v.ID)
	if token := cli.Subscribe(topic, 0, v.onCommand(ctx)); token.Wait() && token.Error() != nil {
		cli.Disconnect(250)
		return token.Error()
	}
	<-ctx.Done()
	close(v.ackCh)
	wg.Wait()
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

func (v *SimulatedVehicle) onDiscovery() func(paho.Client, paho.Message) {
	return func(_ paho.Client, msg paho.Message) {
		if string(msg.Payload()) != "hello" {
			return
		}
		payload, err := json.Marshal(model.Vehicle{
			ID:         v.ID,
			IsV2G:      true,
			Available:  true,
			MaxPower:   10,
			BatteryKWh: 40,
			SoC:        0.8,
		})
		if err != nil {
			log.Printf("%s: marshal discovery: %v", v.ID, err)
			return
		}
		token := v.client.Publish(fmt.Sprintf("v2g/fleet/response/%s", v.ID), 0, false, payload)
		if !token.WaitTimeout(5 * time.Second) {
			log.Printf("%s: discovery publish timeout", v.ID)
			return
		}
		if err := token.Error(); err != nil {
			log.Printf("%s: publish discovery error: %v", v.ID, err)
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
