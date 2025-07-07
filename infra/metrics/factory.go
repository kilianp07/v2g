package metrics

import (
	"github.com/kilianp07/v2g/core/factory"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// init registers built-in metrics sinks.
func init() {
	_ = coremetrics.RegisterMetricsSink("nop", func(map[string]any) (coremetrics.MetricsSink, error) {
		return coremetrics.NopSink{}, nil
	})

	_ = coremetrics.RegisterMetricsSink("prometheus", func(conf map[string]any) (coremetrics.MetricsSink, error) {
		var c struct {
			Port string `json:"prometheus_port"`
		}
		if err := factory.Decode(conf, &c); err != nil {
			return nil, err
		}
		// Port is returned for the HTTP server only; PromSink itself doesn't use it.
		return NewPromSinkWithRegistry(coremetrics.Config{}, prometheus.DefaultRegisterer)
	})

	_ = coremetrics.RegisterMetricsSink("influx", func(conf map[string]any) (coremetrics.MetricsSink, error) {
		var c struct {
			URL    string `json:"url"`
			Token  string `json:"token"`
			Org    string `json:"org"`
			Bucket string `json:"bucket"`
		}
		if err := factory.Decode(conf, &c); err != nil {
			return nil, err
		}
		return NewInfluxSinkWithFallback(c.URL, c.Token, c.Org, c.Bucket), nil
	})
}
