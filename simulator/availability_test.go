package main

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type stubToken struct{ err error }

func (t *stubToken) Wait() bool                       { return true }
func (t *stubToken) WaitTimeout(d time.Duration) bool { return true }
func (t *stubToken) Done() <-chan struct{}            { ch := make(chan struct{}); close(ch); return ch }
func (t *stubToken) Error() error                     { return t.err }

type stubClient struct {
	mu           sync.Mutex
	subs         []string
	pubs         []string
	disconnected int
}

func (c *stubClient) IsConnected() bool      { return c.disconnected == 0 }
func (c *stubClient) IsConnectionOpen() bool { return c.disconnected == 0 }
func (c *stubClient) Connect() paho.Token    { return &stubToken{} }
func (c *stubClient) Disconnect(uint)        { c.mu.Lock(); c.disconnected++; c.mu.Unlock() }
func (c *stubClient) Publish(topic string, qos byte, retained bool, payload interface{}) paho.Token {
	c.mu.Lock()
	c.pubs = append(c.pubs, topic)
	c.mu.Unlock()
	return &stubToken{}
}
func (c *stubClient) Subscribe(topic string, qos byte, cb paho.MessageHandler) paho.Token {
	c.mu.Lock()
	c.subs = append(c.subs, topic)
	c.mu.Unlock()
	return &stubToken{}
}
func (c *stubClient) SubscribeMultiple(map[string]byte, paho.MessageHandler) paho.Token {
	return &stubToken{}
}
func (c *stubClient) Unsubscribe(...string) paho.Token        { return &stubToken{} }
func (c *stubClient) AddRoute(string, paho.MessageHandler)    {}
func (c *stubClient) OptionsReader() paho.ClientOptionsReader { return paho.ClientOptionsReader{} }

func TestHandleAvailabilityConnect(t *testing.T) {
	rng = rand.New(rand.NewSource(1))
	sc := &stubClient{}
	mqttClientFactory = func(b, c string) (paho.Client, error) { return sc, nil }
	defer func() { mqttClientFactory = realMQTTClient }()

	var prof [24]float64
	prof[time.Now().Hour()] = 1
	v := &SimulatedVehicle{
		ID:           "veh1",
		Broker:       "tcp://localhost:1883",
		TopicPrefix:  "test",
		Availability: prof,
	}
	v.handleAvailabilityTick(context.Background())
	if v.client == nil {
		t.Fatalf("expected client to be set")
	}
	if len(sc.subs) != 2 {
		t.Fatalf("expected 2 subscriptions got %d", len(sc.subs))
	}
}

func TestHandleAvailabilityDisconnect(t *testing.T) {
	rng = rand.New(rand.NewSource(1))
	sc := &stubClient{}
	mqttClientFactory = func(b, c string) (paho.Client, error) { return sc, nil }
	defer func() { mqttClientFactory = realMQTTClient }()

	v := &SimulatedVehicle{
		ID:             "veh1",
		Broker:         "tcp://localhost:1883",
		TopicPrefix:    "test",
		DisconnectRate: 1,
		client:         sc,
	}
	v.handleAvailabilityTick(context.Background())
	if v.client != nil {
		t.Fatalf("expected client to be cleared")
	}
	if sc.disconnected == 0 {
		t.Fatalf("expected disconnect to be called")
	}
}
