package telemetry

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/core/dispatch"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	infmqtt "github.com/kilianp07/v2g/infra/mqtt"
)

// Manager collects telemetry from vehicles either via push or polling.
type Manager struct {
	cfg  config.TelemetryConfig
	cli  paho.Client
	sink coremetrics.VehicleStateRecorder
	log  logger.Logger
	disc dispatch.FleetDiscovery

	respCh chan telemetryMessage

	pollReq     prometheus.Counter
	pollResp    prometheus.Counter
	pollTimeout prometheus.Counter
	lastCollect prometheus.Gauge
	latency     prometheus.Histogram
}

type telemetryMessage struct {
	VehicleID string
	Payload   []byte
	Arrived   time.Time
}

// NewManager connects to MQTT and prepares telemetry collection.
func NewManager(mqttCfg infmqtt.Config, cfg config.TelemetryConfig, sink coremetrics.VehicleStateRecorder, disc dispatch.FleetDiscovery) (*Manager, error) {
	opts, err := infmqtt.NewClientOptions(mqttCfg)
	if err != nil {
		return nil, err
	}
	id := mqttCfg.ClientID
	if id != "" {
		id += "-telemetry"
	} else {
		id = "telemetry-" + uuid.NewString()
	}
	opts.SetClientID(id)
	cli := paho.NewClient(opts)
	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	m := &Manager{
		cfg:         cfg,
		cli:         cli,
		sink:        sink,
		log:         logger.New("telemetry"),
		disc:        disc,
		respCh:      make(chan telemetryMessage, 100),
		pollReq:     prometheus.NewCounter(prometheus.CounterOpts{Name: "telemetry_poll_requests_total", Help: "Number of telemetry poll requests"}),
		pollResp:    prometheus.NewCounter(prometheus.CounterOpts{Name: "telemetry_poll_responses_total", Help: "Number of telemetry poll responses"}),
		pollTimeout: prometheus.NewCounter(prometheus.CounterOpts{Name: "telemetry_poll_timeout_total", Help: "Number of telemetry poll timeouts"}),
		lastCollect: prometheus.NewGauge(prometheus.GaugeOpts{Name: "telemetry_last_collect_timestamp_seconds", Help: "Unix timestamp of last telemetry collection"}),
		latency:     prometheus.NewHistogram(prometheus.HistogramOpts{Name: "telemetry_collect_latency_seconds", Help: "Latency of telemetry collection", Buckets: prometheus.DefBuckets}),
	}
	// Register metrics
	prometheus.MustRegister(m.pollReq, m.pollResp, m.pollTimeout, m.lastCollect, m.latency)
	return m, nil
}

// Start runs telemetry collection until context is done.
func (m *Manager) Start(ctx context.Context) {
	mode := strings.ToLower(m.cfg.Mode)
	if mode == "" {
		mode = "push"
	}
	if mode == "push" || mode == "hybrid" {
		topic := strings.TrimSuffix(m.cfg.StatePrefix, "/") + "/+"
		if token := m.cli.Subscribe(topic, 0, m.onPush); token.Wait() && token.Error() != nil {
			m.log.Errorf("subscribe state: %v", token.Error())
		}
	}
	if mode == "pull" || mode == "hybrid" {
		topic := strings.TrimSuffix(m.cfg.ResponsePrefix, "/") + "/+"
		if token := m.cli.Subscribe(topic, 0, m.onResponse); token.Wait() && token.Error() != nil {
			m.log.Errorf("subscribe response: %v", token.Error())
		}
		go m.pollLoop(ctx)
	}
	<-ctx.Done()
	if m.cli.IsConnected() {
		m.cli.Disconnect(250)
	}
}

func (m *Manager) onPush(_ paho.Client, msg paho.Message) {
	if err := m.process(msg.Payload(), msg.Topic(), "push"); err != nil {
		m.log.Errorf("push decode: %v", err)
	}
}

func (m *Manager) onResponse(_ paho.Client, msg paho.Message) {
	m.respCh <- telemetryMessage{VehicleID: extractID(msg.Topic()), Payload: msg.Payload(), Arrived: time.Now()}
}

func extractID(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func (m *Manager) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(m.cfg.Interval()) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.doPoll(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) doPoll(ctx context.Context) {
	start := time.Now()
	var expected map[string]struct{}
	if m.disc != nil {
		exp := make(map[string]struct{})
		vehicles, err := m.disc.Discover(ctx, time.Duration(m.cfg.Timeout())*time.Second)
		if err == nil {
			for _, v := range vehicles {
				exp[v.ID] = struct{}{}
			}
		}
		expected = exp
	} else {
		expected = map[string]struct{}{}
	}
	m.pollReq.Inc()
	token := m.cli.Publish(m.cfg.RequestTopic, 0, false, []byte("poll"))
	token.Wait()
	timeout := time.NewTimer(time.Duration(m.cfg.Timeout()) * time.Second)
	for {
		select {
		case resp := <-m.respCh:
			if err := m.process(resp.Payload, "", "poll"); err != nil {
				m.log.Errorf("poll decode: %v", err)
			} else {
				m.pollResp.Inc()
				m.latency.Observe(time.Since(start).Seconds())
				m.lastCollect.SetToCurrentTime()
				delete(expected, resp.VehicleID)
			}
		case <-timeout.C:
			for range expected {
				m.pollTimeout.Inc()
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) process(payload []byte, topic, context string) error {
	var msg struct {
		VehicleID string  `json:"vehicle_id"`
		SoC       float64 `json:"soc"`
		Available *bool   `json:"available"`
		Charging  *bool   `json:"charging"`
		PowerKW   float64 `json:"power_kw"`
		TS        *int64  `json:"ts"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return err
	}
	if msg.VehicleID == "" {
		msg.VehicleID = extractID(topic)
	}
	ts := time.Now()
	if msg.TS != nil {
		ts = time.Unix(*msg.TS, 0)
	}
	if msg.SoC < 0 {
		msg.SoC = 0
	} else if msg.SoC > 1 {
		msg.SoC = 1
	}
	v := model.Vehicle{ID: msg.VehicleID, SoC: msg.SoC, MaxPower: msg.PowerKW}
	if msg.Available != nil {
		v.Available = *msg.Available
	}
	if msg.Charging != nil {
		v.Charging = *msg.Charging
	}
	if m.sink != nil {
		_ = m.sink.RecordVehicleState(coremetrics.VehicleStateEvent{Vehicle: v, Context: context, Component: "telemetry", Time: ts})
	}
	return nil
}
