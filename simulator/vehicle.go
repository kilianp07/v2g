package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/kilianp07/v2g/metrics"
	"github.com/kilianp07/v2g/model"
)

// SimulatedVehicle connects to MQTT and acknowledges commands.
type command struct {
	id       string
	power    float64
	received time.Time
}

// SimulatedVehicle connects to MQTT and acknowledges commands.
type SimulatedVehicle struct {
	ID          string
	Broker      string
	TopicPrefix string
	Strategy    AckStrategy
	IsV2G       bool
	MaxPower    float64
	Battery     *Battery
	Interval    time.Duration
	Metrics     metrics.MetricsSink

	mu           sync.Mutex
	currentPower float64

	client paho.Client
	ackCh  chan command
}

// NewSimulatedVehicle creates a new vehicle.
func NewSimulatedVehicle(id, broker, prefix string, strat AckStrategy, battery *Battery, interval time.Duration, maxPower float64, sink metrics.MetricsSink) *SimulatedVehicle {
	return &SimulatedVehicle{
		ID:          id,
		Broker:      broker,
		TopicPrefix: prefix,
		Strategy:    strat,
		IsV2G:       true,
		MaxPower:    maxPower,
		Battery:     battery,
		Interval:    interval,
		Metrics:     sink,
		ackCh:       make(chan command, 50),
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		v.batteryLoop(ctx)
	}()
	broadcast := strings.TrimSuffix(v.TopicPrefix, "/") + "/fleet/discovery"
	if token := cli.Subscribe(broadcast, 0, v.onDiscovery()); token.Wait() && token.Error() != nil {
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
			CommandID string  `json:"command_id"`
			PowerKW   float64 `json:"power_kw"`
		}
		if err := json.Unmarshal(msg.Payload(), &m); err != nil {
			log.Printf("%s: decode command: %v", v.ID, err)
			return
		}
		now := time.Now()
		allowed := v.applyPowerOrder(m.PowerKW)
		if rec, ok := v.Metrics.(metrics.DispatchOrderRecorder); ok {
			_ = rec.RecordDispatchOrder(metrics.DispatchOrderEvent{
				DispatchID: m.CommandID,
				VehicleID:  v.ID,
				Signal:     model.SignalFCR,
				PowerKW:    allowed,
				Time:       now,
			})
		}
		cmd := command{id: m.CommandID, power: allowed, received: now}
		select {
		case v.ackCh <- cmd:
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
			IsV2G:      v.IsV2G,
			Available:  true,
			MaxPower:   v.MaxPower,
			BatteryKWh: v.Battery.CapacityKWh,
			SoC:        v.Battery.Soc,
		})
		if err != nil {
			log.Printf("%s: marshal discovery: %v", v.ID, err)
			return
		}
		resp := strings.TrimSuffix(v.TopicPrefix, "/") + "/fleet/response/" + v.ID
		token := v.client.Publish(resp, 0, false, payload)
		if !token.WaitTimeout(5 * time.Second) {
			log.Printf("%s: discovery publish timeout", v.ID)
			return
		}
		if err := token.Error(); err != nil {
			log.Printf("%s: publish discovery error: %v", v.ID, err)
			return
		}
		log.Printf("%s responded to discovery", v.ID)
	}
}

func (v *SimulatedVehicle) worker(ctx context.Context) {
	for {
		select {
		case cmd, ok := <-v.ackCh:
			if !ok {
				return
			}
			sent := v.Strategy.Ack(ctx, v.client, v.ID, cmd.id)
			if rec, ok := v.Metrics.(metrics.DispatchAckRecorder); ok {
				_ = rec.RecordDispatchAck(metrics.DispatchAckEvent{
					DispatchID:   cmd.id,
					VehicleID:    v.ID,
					Signal:       model.SignalFCR,
					Acknowledged: sent,
					Latency:      time.Since(cmd.received),
					Time:         time.Now(),
				})
			}
			_ = v.Metrics.RecordDispatchResult([]metrics.DispatchResult{{
				Signal: model.FlexibilitySignal{
					Type:      model.SignalFCR,
					PowerKW:   cmd.power,
					Timestamp: cmd.received,
				},
				VehicleID:    v.ID,
				PowerKW:      cmd.power,
				Acknowledged: sent,
				DispatchTime: time.Now(),
			}})
		case <-ctx.Done():
			return
		}
	}
}

func (v *SimulatedVehicle) applyPowerOrder(p float64) float64 {
	v.mu.Lock()
	defer v.mu.Unlock()
	if p > 0 {
		if p > v.MaxPower {
			log.Printf("%s: requested power %.1f exceeds max %.1f", v.ID, p, v.MaxPower)
			return v.currentPower
		}
		if p > v.Battery.DischargeRateKW {
			log.Printf("%s: discharge rate %.1f exceeds limit %.1f", v.ID, p, v.Battery.DischargeRateKW)
			return v.currentPower
		}
		if v.Battery.Soc <= 0 {
			log.Printf("%s: battery empty, cannot discharge", v.ID)
			return v.currentPower
		}
	} else if p < 0 {
		ap := -p
		if ap > v.MaxPower {
			log.Printf("%s: requested power %.1f exceeds max %.1f", v.ID, p, v.MaxPower)
			return v.currentPower
		}
		if ap > v.Battery.ChargeRateKW {
			log.Printf("%s: charge rate %.1f exceeds limit %.1f", v.ID, ap, v.Battery.ChargeRateKW)
			return v.currentPower
		}
		if v.Battery.Soc >= 1 {
			log.Printf("%s: battery full, cannot charge", v.ID)
			return v.currentPower
		}
	}
	v.currentPower = p
	return p
}

func (v *SimulatedVehicle) batteryLoop(ctx context.Context) {
	ticker := time.NewTicker(v.Interval)
	defer ticker.Stop()
	last := time.Now()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			dt := now.Sub(last)
			last = now
			v.mu.Lock()
			applied := v.Battery.ApplyPower(v.currentPower, dt)
			v.currentPower = applied
			soc := v.Battery.Soc
			v.mu.Unlock()
			if rec, ok := v.Metrics.(metrics.VehicleStateRecorder); ok {
				_ = rec.RecordVehicleState(metrics.VehicleStateEvent{
					Vehicle: model.Vehicle{
						ID:         v.ID,
						IsV2G:      v.IsV2G,
						Available:  true,
						MaxPower:   v.MaxPower,
						BatteryKWh: v.Battery.CapacityKWh,
						SoC:        soc,
					},
					Context:   "simulation",
					Component: "simulator",
					Time:      now,
				})
			}
			state, _ := json.Marshal(struct {
				SoC float64 `json:"soc"`
			}{SoC: soc})
			t := v.client.Publish(fmt.Sprintf("vehicle/state/%s", v.ID), 0, false, state)
			if !t.WaitTimeout(5 * time.Second) {
				log.Printf("%s: state publish timeout", v.ID)
			}
			_ = t.Error()
		case <-ctx.Done():
			return
		}
	}
}
