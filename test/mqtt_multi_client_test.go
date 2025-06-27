package test

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/mqtt"
)

// TestMultipleMQTTClients ensures dispatcher and discovery clients can operate
// concurrently using unique client IDs.
func TestMultipleMQTTClients(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}

	ctx := context.Background()
	cont, broker := startMosquitto(ctx, t)
	defer func() { _ = cont.Terminate(ctx) }()

	sim := startSimulator(t, broker)
	defer sim.Disconnect(100)

	pub := newPublisher(t, broker)
	disc := newDiscovery(t, broker)
	defer func() {
		if err := disc.Close(); err != nil {
			t.Logf("close discovery: %v", err)
		}
	}()

	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second)
	vehicles, err := disc.Discover(ctxTimeout, time.Second)
	cancel()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(vehicles) != 1 || vehicles[0].ID != "veh1" {
		t.Fatalf("unexpected vehicles: %+v", vehicles)
	}

	cmdID, err := pub.SendOrder("veh1", 10)
	if err != nil {
		t.Fatalf("send order: %v", err)
	}
	ack, err := pub.WaitForAck(cmdID, time.Second)
	if err != nil || !ack {
		t.Fatalf("ack failed: %v", err)
	}
}

func startSimulator(t *testing.T, broker string) paho.Client {
	t.Helper()
	simOpts := paho.NewClientOptions().AddBroker(broker).SetClientID("simulator")
	sim := paho.NewClient(simOpts)
	if token := sim.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("sim connect: %v", token.Error())
	}
	if token := sim.Subscribe("v2g/fleet/discovery", 0, func(_ paho.Client, m paho.Message) {
		if string(m.Payload()) == "hello" {
			payload, _ := json.Marshal(model.Vehicle{ID: "veh1", IsV2G: true, Available: true})
			sim.Publish("v2g/fleet/response/veh1", 0, false, payload)
		}
	}); token.Wait() && token.Error() != nil {
		t.Fatalf("sim subscribe discovery: %v", token.Error())
	}
	if token := sim.Subscribe("vehicle/veh1/command", 0, func(_ paho.Client, m paho.Message) {
		var cmd struct {
			CommandID string `json:"command_id"`
		}
		_ = json.Unmarshal(m.Payload(), &cmd)
		payload, _ := json.Marshal(map[string]string{"command_id": cmd.CommandID})
		sim.Publish("vehicle/veh1/ack", 0, false, payload)
	}); token.Wait() && token.Error() != nil {
		t.Fatalf("sim subscribe command: %v", token.Error())
	}
	return sim
}

func newPublisher(t *testing.T, broker string) *mqtt.PahoClient {
	t.Helper()
	pubCfg := mqtt.Config{Broker: broker, ClientID: "dispatcher", AckTopic: "vehicle/+/ack"}
	pub, err := mqtt.NewPahoClient(pubCfg)
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}
	return pub
}

func newDiscovery(t *testing.T, broker string) *mqtt.PahoFleetDiscovery {
	t.Helper()
	discCfg := mqtt.Config{Broker: broker, ClientID: "dispatcher"}
	disc, err := mqtt.NewPahoFleetDiscovery(discCfg, "v2g/fleet/discovery", "v2g/fleet/response/+", "hello")
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	return disc
}
