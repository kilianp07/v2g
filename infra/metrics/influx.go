package metrics

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
)

// InfluxSink writes dispatch events to an InfluxDB instance using the official client.
type InfluxSink struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	log      logger.Logger
}

// NewInfluxSink creates a new sink configured for the given InfluxDB endpoint.
func NewInfluxSink(url, token, org, bucket string) *InfluxSink {
	base := strings.TrimSuffix(url, "/api/v2/write")
	client := influxdb2.NewClientWithOptions(base, token,
		influxdb2.DefaultOptions().SetHTTPClient(&http.Client{Timeout: 5 * time.Second}))
	return &InfluxSink{
		client:   client,
		writeAPI: client.WriteAPIBlocking(org, bucket),
		log:      logger.New("influx-sink"),
	}
}

// NewInfluxSinkWithFallback tries to ping the InfluxDB instance and
// returns a NopSink if the health check fails.
func NewInfluxSinkWithFallback(url, token, org, bucket string) coremetrics.MetricsSink {
	sink := NewInfluxSink(url, token, org, bucket)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	health, err := sink.client.Health(ctx)
	if err != nil || health.Status != "pass" {
		if err != nil {
			sink.log.Errorf("influx health check error: %v", err)
		} else {
			sink.log.Errorf("influx health status: %s", health.Status)
		}
		sink.client.Close()
		return coremetrics.NopSink{}
	}
	return sink
}

// RecordDispatchResult writes the dispatch result as line protocol events.
func (s *InfluxSink) RecordDispatchResult(res []coremetrics.DispatchResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, r := range res {
		p := write.NewPointWithMeasurement("dispatch_event").
			AddTag("vehicle_id", r.VehicleID).
			AddTag("signal_type", signalToString(r.Signal.Type)).
			AddTag("acknowledged", strconv.FormatBool(r.Acknowledged)).
			AddTag("dispatch_id", strconv.FormatInt(r.Signal.Timestamp.UnixNano(), 10)).
			AddTag("component", "dispatch_manager").
			AddField("power_kw", round3(r.PowerKW)).
			AddField("score", round3(r.Score)).
			AddField("market_price", round3(r.MarketPrice)).
			SetTime(r.Signal.Timestamp)
		if err := s.writeAPI.WritePoint(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// RecordFleetDiscovery persists the result of a discovery cycle.
func (s *InfluxSink) RecordFleetDiscovery(ev coremetrics.FleetDiscoveryEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p := write.NewPointWithMeasurement("fleet_discovery_event").
		AddTag("component", ev.Component).
		AddField("pings", ev.Pings).
		AddField("responses", ev.Responses).
		SetTime(ev.Time)
	return s.writeAPI.WritePoint(ctx, p)
}

// RecordVehicleState writes a snapshot of a vehicle.
func (s *InfluxSink) RecordVehicleState(ev coremetrics.VehicleStateEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	v := ev.Vehicle
	p := write.NewPointWithMeasurement("vehicle_state").
		AddTag("vehicle_id", v.ID)
	if ev.Component != "" {
		p.AddTag("component", ev.Component)
	}
	p = p.AddTag("context", ev.Context).
		AddField("soc", round3(v.SoC)).
		AddField("available", v.Available).
		AddField("charging", v.Charging).
		AddField("power_kw", round3(v.MaxPower)).
		SetTime(ev.Time)
	return s.writeAPI.WritePoint(ctx, p)
}

// RecordDispatchOrder records an order being sent.
func (s *InfluxSink) RecordDispatchOrder(ev coremetrics.DispatchOrderEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p := write.NewPointWithMeasurement("dispatch_order_sent").
		AddTag("vehicle_id", ev.VehicleID).
		AddTag("signal_type", signalToString(ev.Signal)).
		AddTag("dispatch_id", ev.DispatchID).
		AddTag("component", "dispatch_manager").
		AddField("power_kw", round3(ev.PowerKW)).
		AddField("score", round3(ev.Score)).
		AddField("market_price", round3(ev.MarketPrice)).
		SetTime(ev.Time)
	return s.writeAPI.WritePoint(ctx, p)
}

// RecordDispatchAck records an acknowledgment result.
func (s *InfluxSink) RecordDispatchAck(ev coremetrics.DispatchAckEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p := write.NewPointWithMeasurement("dispatch_ack_received").
		AddTag("vehicle_id", ev.VehicleID).
		AddTag("signal_type", signalToString(ev.Signal)).
		AddTag("acknowledged", strconv.FormatBool(ev.Acknowledged)).
		AddTag("dispatch_id", ev.DispatchID).
		AddTag("component", "dispatch_manager").
		AddField("latency_ms", round3(ev.Latency.Seconds()*1000)).
		AddField("errors", ev.Error).
		SetTime(ev.Time)
	return s.writeAPI.WritePoint(ctx, p)
}

// RecordFallback records a fallback application.
func (s *InfluxSink) RecordFallback(ev coremetrics.FallbackEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p := write.NewPointWithMeasurement("fallback_applied").
		AddTag("dispatch_id", ev.DispatchID).
		AddTag("signal_type", signalToString(ev.Signal)).
		AddTag("component", "fallback")
	if ev.VehicleID != "" {
		p = p.AddTag("vehicle_id", ev.VehicleID)
	}
	p = p.AddField("power_kw", round3(ev.ResidualPower)).
		AddField("fallback_reason", ev.Reason).
		SetTime(ev.Time)
	return s.writeAPI.WritePoint(ctx, p)
}

// RecordRTESignal writes a received flexibility signal.
func (s *InfluxSink) RecordRTESignal(ev coremetrics.RTESignalEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p := write.NewPointWithMeasurement("rte_signal_received").
		AddTag("signal_type", signalToString(ev.Signal.Type)).
		AddTag("component", "rte_connector").
		AddField("power_kw", round3(ev.Signal.PowerKW)).
		SetTime(ev.Time)
	return s.writeAPI.WritePoint(ctx, p)
}

// LogVehicleState is a helper to record a vehicle snapshot with a context tag.
func (s *InfluxSink) LogVehicleState(v model.Vehicle, context string) error {
	return s.RecordVehicleState(coremetrics.VehicleStateEvent{Vehicle: v, Context: context, Time: time.Now()})
}

func round3(f float64) float64 {
	return math.Round(f*1000) / 1000
}

func signalToString(t model.SignalType) string {
	return t.String()
}
