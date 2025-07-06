package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	coremqtt "github.com/kilianp07/v2g/core/mqtt"
	"github.com/kilianp07/v2g/infra/logger"
)

// Config defines the connection parameters for the Paho MQTT client.
type Config struct {
	Broker     string          `json:"broker"`
	ClientID   string          `json:"client_id"`
	Username   string          `json:"username"`
	Password   string          `json:"password"`
	AckTopic   string          `json:"ack_topic"`
	UseTLS     bool            `json:"use_tls"`
	ClientCert string          `json:"client_cert"`
	ClientKey  string          `json:"client_key"`
	CABundle   string          `json:"ca_bundle"`
	AuthMethod string          `json:"auth_method"`
	QoS        map[string]byte `json:"qos"`
	LWTTopic   string          `json:"lwt_topic"`
	LWTPayload string          `json:"lwt_payload"`
	LWTQoS     byte            `json:"lwt_qos"`
	LWTRetain  bool            `json:"lwt_retain"`
	MaxRetries int             `json:"max_retries"`
	BackoffMS  int             `json:"backoff_ms"`
	TLSConfig  *tls.Config     `json:"-"`
}

// PahoClient implements the Publisher interface using Eclipse Paho.
type pahoClient interface {
	IsConnected() bool
	Connect() paho.Token
	Disconnect(quiesce uint)
	Publish(topic string, qos byte, retained bool, payload interface{}) paho.Token
	Subscribe(topic string, qos byte, callback paho.MessageHandler) paho.Token
}

type PahoClient struct {
	cli      pahoClient
	ackTopic string
	qos      map[string]byte

	mu         sync.Mutex
	ackChans   map[string]chan struct{}
	logger     logger.Logger
	lwtTopic   string
	lwtPayload string
	lwtQoS     byte
	lwtRetain  bool
	maxRetries int
	backoff    time.Duration
}

var newMQTTClient = func(opts *paho.ClientOptions) pahoClient {
	return paho.NewClient(opts)
}

// NewPahoClient connects to the MQTT broker and subscribes to the ACK topic.
func NewPahoClient(cfg Config) (*PahoClient, error) {
	opts, err := NewClientOptions(cfg)
	if err != nil {
		return nil, err
	}

	logger := logger.New("mqtt_client")
	pc := &PahoClient{ackTopic: cfg.AckTopic,
		ackChans:   make(map[string]chan struct{}),
		logger:     logger,
		qos:        cfg.QoS,
		lwtTopic:   cfg.LWTTopic,
		lwtPayload: cfg.LWTPayload,
		lwtQoS:     cfg.LWTQoS,
		lwtRetain:  cfg.LWTRetain,
		maxRetries: cfg.MaxRetries,
		backoff:    time.Duration(cfg.BackoffMS) * time.Millisecond,
	}

	opts.OnConnect = func(c paho.Client) {
		logger.Infof("MQTT connected")
		qos := byte(0)
		if q, ok := pc.qos["ack"]; ok {
			qos = q
		}
		if token := c.Subscribe(pc.ackTopic, qos, pc.onAck); token.Wait() && token.Error() != nil {
			logger.Errorf("subscribe error: %v", token.Error())
		}
	}
	opts.OnConnectionLost = func(_ paho.Client, err error) {
		logger.Errorf("connection lost: %v", err)
	}
	opts.OnReconnecting = func(_ paho.Client, _ *paho.ClientOptions) {
		logger.Warnf("reconnecting to MQTT broker")
	}
	c := newMQTTClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	pc.cli = c
	return pc, nil
}

// NewClientOptions builds mqtt client options from Config.
func NewClientOptions(cfg Config) (*paho.ClientOptions, error) {
	opts := paho.NewClientOptions().AddBroker(cfg.Broker).SetClientID(cfg.ClientID)
	opts.AutoReconnect = true
	if cfg.AuthMethod == "username_password" || cfg.AuthMethod == "both" || cfg.AuthMethod == "" {
		if cfg.Username != "" {
			opts.SetUsername(cfg.Username)
		}
		if cfg.Password != "" {
			opts.SetPassword(cfg.Password)
		}
	}
	if cfg.UseTLS {
		tlsCfg, err := cfg.LoadTLSConfig()
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsCfg)
	}
	if cfg.LWTTopic != "" {
		opts.SetWill(cfg.LWTTopic, cfg.LWTPayload, cfg.LWTQoS, cfg.LWTRetain)
	}
	return opts, nil
}

// LoadTLSConfig loads the TLS configuration from the file paths in the config.
func (c Config) LoadTLSConfig() (*tls.Config, error) {
	if c.TLSConfig != nil {
		return c.TLSConfig, nil
	}
	if c.ClientCert == "" || c.ClientKey == "" || c.CABundle == "" {
		return nil, fmt.Errorf("tls config requires client_cert, client_key and ca_bundle")
	}
	cert, err := tls.LoadX509KeyPair(c.ClientCert, c.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("load cert: %w", err)
	}
	caBytes, err := os.ReadFile(c.CABundle)
	if err != nil {
		return nil, fmt.Errorf("read ca: %w", err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caBytes)
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: pool, MinVersion: tls.VersionTLS12}
	return cfg, nil
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
	qos := byte(0)
	if q, ok := p.qos["command"]; ok {
		qos = q
	}
	if p.maxRetries <= 0 {
		p.maxRetries = 3
	}
	if p.backoff <= 0 {
		p.backoff = 100 * time.Millisecond
	}
	var publishErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		token := p.cli.Publish(topic, qos, false, payload)
		token.Wait()
		publishErr = token.Error()
		if publishErr == nil {
			p.logger.Infof("sent order %s to %s", cmdID, topic)
			break
		}
		p.logger.Errorf("publish attempt %d failed: %v", attempt+1, publishErr)
		time.Sleep(p.backoff * time.Duration(1<<attempt))
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
		return false, fmt.Errorf("%w", coremqtt.ErrAckTimeout)
	}
}

// Disconnect gracefully closes the MQTT connection.
func (p *PahoClient) Disconnect() {
	if p.cli != nil && p.cli.IsConnected() {
		p.cli.Disconnect(250)
	}
}
