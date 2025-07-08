package mqtt

import (
	"fmt"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	coremon "github.com/kilianp07/v2g/core/monitoring"
)

type recordMonitor struct {
	err  error
	tags map[string]string
}

func (r *recordMonitor) CaptureException(err error, tags map[string]string) {
	r.err = err
	r.tags = tags
}
func (r *recordMonitor) Recover()            {}
func (r *recordMonitor) Flush(time.Duration) {}

func TestSendOrderErrorCaptured(t *testing.T) {
	mc := &mockClient{publishErrs: []error{fmt.Errorf("net fail"), fmt.Errorf("net fail"), fmt.Errorf("net fail"), fmt.Errorf("net fail")}}
	newMQTTClient = func(o *paho.ClientOptions) pahoClient { mc.opts = o; return mc }
	defer func() { newMQTTClient = func(opts *paho.ClientOptions) pahoClient { return paho.NewClient(opts) } }()
	mon := &recordMonitor{}
	coremon.Init(mon)
	cfg := Config{Broker: "tcp://localhost:1883", ClientID: "id", AckTopic: "a", MaxRetries: 0, BackoffMS: 1}
	cli, err := NewPahoClient(cfg)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	_, err = cli.SendOrder("veh1", 1)
	if err == nil {
		t.Fatalf("expected error")
	}
	if mon.err == nil {
		t.Fatalf("error not captured")
	}
	if mon.tags["vehicle_id"] != "veh1" || mon.tags["module"] != "mqtt" {
		t.Fatalf("tags not set")
	}
	coremon.Init(coremon.NopMonitor{})
}
