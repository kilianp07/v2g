package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/dispatch"
	"github.com/kilianp07/v2g/logger"
	"github.com/kilianp07/v2g/metrics"
	"github.com/kilianp07/v2g/mqtt"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logg := logger.New("main")
	zlog := zerolog.New(os.Stdout).With().Timestamp().Str("component", "metrics").Logger()
	mqttCfg := cfg.MQTT
	mqttCfg.Logger = logg
	client, err := mqtt.NewPahoClient(mqttCfg)
	if err != nil {
		log.Fatalf("mqtt client: %v", err)
	}

	var sinks []metrics.MetricsSink
	if cfg.Metrics.PrometheusEnabled {
		sink, err := metrics.NewPromSink(cfg.Metrics)
		if err != nil {
			log.Fatalf("prom sink: %v", err)
		}
		sinks = append(sinks, sink)
		go func() {
			if err := metrics.StartPromServer(ctx, cfg.Metrics.PrometheusPort); err != nil {
				logg.Errorf("prom server: %v", err)
			}
		}()
	}
	if cfg.Metrics.InfluxEnabled {
		sink := metrics.NewInfluxSinkWithFallback(cfg.Metrics, &zlog)
		sinks = append(sinks, sink)
	}
	var sink metrics.MetricsSink = metrics.NopSink{}
	if len(sinks) == 1 {
		sink = sinks[0]
	} else if len(sinks) > 1 {
		sink = metrics.NewMultiSink(sinks...)
	}

	ackTimeout := time.Duration(cfg.Dispatch.AckTimeoutSeconds) * time.Second
	manager, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatch.EqualDispatcher{},
		dispatch.NoopFallback{},
		client,
		ackTimeout,
		logg,
		sink,
	)
	if err != nil {
		log.Fatalf("dispatch manager: %v", err)
	}
	_ = manager

	<-ctx.Done()
}
