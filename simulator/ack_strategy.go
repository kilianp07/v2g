package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// AckStrategy defines how a vehicle acknowledges commands.
type AckStrategy interface {
	// Ack publishes an acknowledgment. It returns true if the ACK was sent.
	Ack(ctx context.Context, cli paho.Client, vehicleID, commandID string) bool
}

// AutoAck sends an ACK after an optional fixed delay.
type AutoAck struct {
	Delay time.Duration
}

// Ack implements AckStrategy.
func (a AutoAck) Ack(ctx context.Context, cli paho.Client, vehicleID, commandID string) bool {
	if a.Delay > 0 {
		select {
		case <-time.After(a.Delay):
		case <-ctx.Done():
			return false
		}
	}
	return publishAck(cli, vehicleID, commandID)
}

// RandomAck drops acknowledgments with the configured probability and
// waits for the specified delay before sending.
type RandomAck struct {
	Delay    time.Duration
	DropRate float64
}

// Ack implements AckStrategy.
func (r RandomAck) Ack(ctx context.Context, cli paho.Client, vehicleID, commandID string) bool {
	if r.DropRate > 0 && rng.Float64() < r.DropRate {
		return false
	}
	if r.Delay > 0 {
		select {
		case <-time.After(r.Delay):
		case <-ctx.Done():
			return false
		}
	}
	return publishAck(cli, vehicleID, commandID)
}

func publishAck(cli paho.Client, vehicleID, commandID string) bool {
	payload, err := json.Marshal(struct {
		CommandID string `json:"command_id"`
	}{CommandID: commandID})
	if err != nil {
		log.Printf("marshal ack: %v", err)
		return false
	}
	token := cli.Publish(fmt.Sprintf("vehicle/%s/ack", vehicleID), 0, false, payload)
	if !token.WaitTimeout(5 * time.Second) {
		log.Printf("ack publish timeout for %s", vehicleID)
		return false
	}
	if err := token.Error(); err != nil {
		log.Printf("publish ack error for %s: %v", vehicleID, err)
		return false
	}
	log.Printf("%s: published ack %s", vehicleID, commandID)
	return true
}
