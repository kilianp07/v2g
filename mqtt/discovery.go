package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

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

// Close disconnects the underlying MQTT client.
func (d *PahoFleetDiscovery) Close() error {
	if d.cli != nil && d.cli.IsConnected() {
		d.cli.Disconnect(250)
	}
	return nil
}

// NewPahoFleetDiscovery connects to the broker and returns a discovery instance.
func NewPahoFleetDiscovery(cfg Config, broadcastTopic, responseTopic, magicWord string) (*PahoFleetDiscovery, error) {
	id := cfg.ClientID
	if id != "" {
		id += "-discovery"
	} else {
		id = "discovery-" + uuid.NewString()
	}
	opts := paho.NewClientOptions().AddBroker(cfg.Broker).SetClientID(id)
	opts.AutoReconnect = true
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
		cfg.Password = ""
	}
	if cfg.UseTLS {
		if cfg.TLSConfig == nil {
			return nil, fmt.Errorf("mqtt tls enabled but TLSConfig is nil")
		}
		if len(cfg.TLSConfig.Certificates) == 0 && cfg.TLSConfig.GetCertificate == nil {
			return nil, fmt.Errorf("tls config missing certificate")
		}
		opts.SetTLSConfig(cfg.TLSConfig)
	}

	d := &PahoFleetDiscovery{
		broadcastTopic: broadcastTopic,
		responseTopic:  responseTopic,
		magicWord:      magicWord,
		log:            logger.New("fleet_discovery"),
	}
	opts.OnConnect = func(c paho.Client) {
		d.log.Infof("MQTT connected as %s", id)
	}
	opts.OnConnectionLost = func(_ paho.Client, err error) {
		d.log.Errorf("connection lost: %v", err)
	}
	opts.OnReconnecting = func(_ paho.Client, _ *paho.ClientOptions) {
		d.log.Warnf("reconnecting to MQTT broker")
	}
	cli := paho.NewClient(opts)
	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("mqtt connect %s: %w", cfg.Broker, token.Error())
	}
	d.cli = cli
	return d, nil
}

// Discover broadcasts the magic word and collects vehicle responses until the timeout.
func (d *PahoFleetDiscovery) Discover(ctx context.Context, timeout time.Duration) ([]model.Vehicle, error) {
	var (
		vehicles []model.Vehicle
		errs     []error
		mu       sync.Mutex
		ids      = make(map[string]struct{})
	)
	handler := func(_ paho.Client, m paho.Message) {
		var v model.Vehicle
		if err := json.Unmarshal(m.Payload(), &v); err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("invalid discovery payload: %w", err))
			mu.Unlock()
			return
		}
		d.log.Debugf("received discovery response from %s", v.ID)
		mu.Lock()
		if _, exists := ids[v.ID]; !exists {
			ids[v.ID] = struct{}{}
			vehicles = append(vehicles, v)
		}
		mu.Unlock()
	}

	if token := d.cli.Subscribe(d.responseTopic, 0, handler); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	d.log.Debugf("subscribed to %s", d.responseTopic)

	// publish broadcast
	d.log.Debugf("publishing discovery ping")
	if token := d.cli.Publish(d.broadcastTopic, 0, false, []byte(d.magicWord)); token.Wait() && token.Error() != nil {
		_ = d.cli.Unsubscribe(d.responseTopic)
		return nil, token.Error()
	}

	timer := time.NewTimer(timeout)
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	timer.Stop()

	if token := d.cli.Unsubscribe(d.responseTopic); token.Wait() && token.Error() != nil {
		d.log.Errorf("unsubscribe error: %v", token.Error())
		if t := d.cli.Unsubscribe(d.responseTopic); t.Wait() && t.Error() != nil {
			d.log.Errorf("retry unsubscribe error: %v", t.Error())
		}
	}
	mu.Lock()
	err := errors.Join(errs...)
	res := append([]model.Vehicle(nil), vehicles...)
	mu.Unlock()
	d.log.Infof("discovered %d vehicles", len(res))
	return res, err
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

func (m MockDiscovery) Close() error { return nil }
