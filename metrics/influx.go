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

	"github.com/kilianp07/v2g/logger"

	"github.com/kilianp07/v2g/model"
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
func NewInfluxSinkWithFallback(cfg Config) MetricsSink {
	sink := NewInfluxSink(cfg.InfluxURL, cfg.InfluxToken, cfg.InfluxOrg, cfg.InfluxBucket)
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
		return NopSink{}
	}
	return sink
}

// RecordDispatchResult writes the dispatch result as line protocol events.
func (s *InfluxSink) RecordDispatchResult(res []DispatchResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, r := range res {
		p := write.NewPointWithMeasurement("dispatch_event").
			AddTag("vehicle_id", r.VehicleID).
			AddTag("signal_type", signalToString(r.Signal.Type)).
			AddTag("acknowledged", strconv.FormatBool(r.Acknowledged)).
			AddField("power_kw", round3(r.PowerKW)).
			AddField("score", round3(r.Score)).
			AddField("market_price", round3(r.MarketPrice)).
			AddField("dispatch_time", r.DispatchTime.UnixNano()).
			SetTime(r.Signal.Timestamp)
		if err := s.writeAPI.WritePoint(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func round3(f float64) float64 {
	return math.Round(f*1000) / 1000
}

func signalToString(t model.SignalType) string {
	return t.String()
}
