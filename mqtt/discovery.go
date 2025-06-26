package mqtt

import (
	"context"
	"encoding/json"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/kilianp07/v2g/logger"
	"github.com/kilianp07/v2g/model"
)

// PahoFleetDiscovery implements dispatch.FleetDiscovery using MQTT broadcast.
// It publishes a magic word on a broadcast topic and collects vehicle states
// from a response topic for a short period.
type PahoFleetDiscovery struct {
	cli            paho.Client
	broadcastTopic string
	responseTopic  string
	magicWord      string
	log            logger.Logger
}

// NewPahoFleetDiscovery connects to the broker and returns a discovery instance.
func NewPahoFleetDiscovery(cfg Config, broadcastTopic, responseTopic, magicWord string) (*PahoFleetDiscovery, error) {
	opts := paho.NewClientOptions().AddBroker(cfg.Broker).SetClientID(cfg.ClientID)
	opts.AutoReconnect = true
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}
	if cfg.UseTLS && cfg.TLSConfig != nil {
		opts.SetTLSConfig(cfg.TLSConfig)
	}

	d := &PahoFleetDiscovery{
		broadcastTopic: broadcastTopic,
		responseTopic:  responseTopic,
		magicWord:      magicWord,
		log:            logger.New("fleet_discovery"),
	}
	opts.OnConnectionLost = func(_ paho.Client, err error) {
		d.log.Errorf("connection lost: %v", err)
	}
	cli := paho.NewClient(opts)
	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	d.cli = cli
	return d, nil
}

// Discover broadcasts the magic word and collects vehicle responses until the timeout.
func (d *PahoFleetDiscovery) Discover(ctx context.Context, timeout time.Duration) ([]model.Vehicle, error) {
	var (
		vehicles []model.Vehicle
		mu       = make(chan model.Vehicle, 16)
	)
	// subscribe
	if token := d.cli.Subscribe(d.responseTopic, 0, func(_ paho.Client, m paho.Message) {
		var v model.Vehicle
		if err := json.Unmarshal(m.Payload(), &v); err != nil {
			d.log.Errorf("invalid discovery payload: %v", err)
			return
		}
		select {
		case mu <- v:
		default:
		}
	}); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	// publish broadcast
	if token := d.cli.Publish(d.broadcastTopic, 0, false, []byte(d.magicWord)); token.Wait() && token.Error() != nil {
		_ = d.cli.Unsubscribe(d.responseTopic)
		return nil, token.Error()
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
loop:
	for {
		select {
		case v := <-mu:
			vehicles = append(vehicles, v)
		case <-ctx.Done():
			break loop
		case <-timer.C:
			break loop
		}
	}

	if token := d.cli.Unsubscribe(d.responseTopic); token.Wait() && token.Error() != nil {
		d.log.Errorf("unsubscribe error: %v", token.Error())
	}
	return vehicles, nil
}

// MockDiscovery is a simple FleetDiscovery used in tests.
type MockDiscovery struct {
	Vehicles []model.Vehicle
}

func (m MockDiscovery) Discover(ctx context.Context, timeout time.Duration) ([]model.Vehicle, error) {
	_ = ctx
	_ = timeout
	return m.Vehicles, nil
}
