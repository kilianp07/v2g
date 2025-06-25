package mqtt

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	"github.com/kilianp07/v2g/logger"
)

// Config defines the connection parameters for the Paho MQTT client.
type Config struct {
	Broker    string      `json:"broker"`
	ClientID  string      `json:"client_id"`
	Username  string      `json:"username"`
	Password  string      `json:"password"`
	AckTopic  string      `json:"ack_topic"`
	UseTLS    bool        `json:"use_tls"`
	TLSConfig *tls.Config `json:"-"`
}

// PahoClient implements the Publisher interface using Eclipse Paho.
type PahoClient struct {
	cli      paho.Client
	ackTopic string

	mu       sync.Mutex
	ackChans map[string]chan struct{}
	logger   logger.Logger
}

// NewPahoClient connects to the MQTT broker and subscribes to the ACK topic.
func NewPahoClient(cfg Config) (*PahoClient, error) {
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

	logger := logger.New("mqtt_client")
	pc := &PahoClient{ackTopic: cfg.AckTopic,
		ackChans: make(map[string]chan struct{}),
		logger:   logger,
	}

	opts.OnConnect = func(c paho.Client) {
		logger.Infof("MQTT connected")
		if token := c.Subscribe(pc.ackTopic, 0, pc.onAck); token.Wait() && token.Error() != nil {
			logger.Errorf("subscribe error: %v", token.Error())
		}
	}
	opts.OnConnectionLost = func(_ paho.Client, err error) {
		logger.Errorf("connection lost: %v", err)
	}
	c := paho.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	pc.cli = c
	return pc, nil
}

func (p *PahoClient) onAck(_ paho.Client, msg paho.Message) {
	var m struct {
		CommandID string `json:"command_id"`
	}
	if err := json.Unmarshal(msg.Payload(), &m); err != nil {
		p.logger.Errorf("failed to decode ack: %v", err)
		return
	}
	p.mu.Lock()
	ch, ok := p.ackChans[m.CommandID]
	if ok {
		select {
		case ch <- struct{}{}:
		default:
		}
		p.logger.Infof("received ack %s", m.CommandID)
	}
	p.mu.Unlock()
}

// SendOrder sends a dispatch order to the vehicle specific topic and returns
// the command identifier used for acknowledgment tracking.
func (p *PahoClient) SendOrder(vehicleID string, powerKW float64) (string, error) {
	cmdID := uuid.NewString()
	order := struct {
		CommandID string  `json:"command_id"`
		VehicleID string  `json:"vehicle_id"`
		PowerKW   float64 `json:"power_kw"`
		Timestamp int64   `json:"timestamp"`
	}{
		CommandID: cmdID,
		VehicleID: vehicleID,
		PowerKW:   powerKW,
		Timestamp: time.Now().UnixMilli(),
	}
	payload, err := json.Marshal(order)
	if err != nil {
		return "", err
	}

	topic := fmt.Sprintf("vehicle/%s/command", vehicleID)
	var publishErr error
	for attempt := 0; attempt < 3; attempt++ {
		token := p.cli.Publish(topic, 0, false, payload)
		token.Wait()
		publishErr = token.Error()
		if publishErr == nil {
			p.logger.Infof("sent order %s to %s", cmdID, topic)
			break
		}
		p.logger.Errorf("publish attempt %d failed: %v", attempt+1, publishErr)
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	if publishErr != nil {
		return "", publishErr
	}

	p.mu.Lock()
	p.ackChans[cmdID] = make(chan struct{}, 1)
	p.mu.Unlock()

	return cmdID, nil
}

// WaitForAck blocks until an ACK for the given command ID is received or timeout.
func (p *PahoClient) WaitForAck(commandID string, timeout time.Duration) (bool, error) {
	p.mu.Lock()
	ch := p.ackChans[commandID]
	p.mu.Unlock()
	if ch == nil {
		return false, fmt.Errorf("unknown command")
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ch:
		p.mu.Lock()
		delete(p.ackChans, commandID)
		p.mu.Unlock()
		return true, nil
	case <-timer.C:
		p.mu.Lock()
		delete(p.ackChans, commandID)
		p.mu.Unlock()
		return false, fmt.Errorf("timeout waiting for ack")
	}
}
