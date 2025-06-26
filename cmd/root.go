package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/dispatch"
	"github.com/kilianp07/v2g/internal/eventbus"
	"github.com/kilianp07/v2g/logger"
	"github.com/kilianp07/v2g/metrics"
	"github.com/kilianp07/v2g/model"
	"github.com/kilianp07/v2g/mqtt"
	"github.com/kilianp07/v2g/rte"
)

var cfgPath string

var rootCmd = &cobra.Command{
	Use:   "v2g",
	Short: "V2G dispatch service",
	RunE:  run,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "config.yaml", "configuration file")
}

// Execute runs the CLI.
func Execute() error { return rootCmd.Execute() }

func run(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logg := logger.New("main")

	client, err := mqtt.NewPahoClient(cfg.MQTT)
	if err != nil {
		return fmt.Errorf("mqtt client: %w", err)
	}

	var sinks []metrics.MetricsSink
	if cfg.Metrics.PrometheusEnabled {
		sink, err := metrics.NewPromSink(cfg.Metrics)
		if err != nil {
			return fmt.Errorf("prom sink: %w", err)
		}
		sinks = append(sinks, sink)
		go func() {
			if err := metrics.StartPromServer(ctx, cfg.Metrics.PrometheusPort); err != nil {
				logg.Errorf("prom server: %v", err)
			}
		}()
	}

	if cfg.Metrics.InfluxEnabled {
		sink := metrics.NewInfluxSinkWithFallback(cfg.Metrics)
		sinks = append(sinks, sink)
	}

	var sink metrics.MetricsSink
	if len(sinks) == 1 {
		sink = sinks[0]
	} else if len(sinks) > 1 {
		sink = metrics.NewMultiSink(sinks...)
	}

	ackTimeout := time.Duration(cfg.Dispatch.AckTimeoutSeconds) * time.Second
	bus := eventbus.New()
	disc, err := mqtt.NewPahoFleetDiscovery(cfg.MQTT, "v2g/fleet/discovery", "v2g/fleet/response/+", "hello")
	if err != nil {
		return fmt.Errorf("fleet discovery: %w", err)
	}
	manager, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatch.EqualDispatcher{},
		dispatch.NoopFallback{},
		client,
		ackTimeout,
		sink,
		bus,
		disc,
	)
	if err != nil {
		return fmt.Errorf("dispatch manager: %w", err)
	}
	defer func() {
		if err := manager.Close(); err != nil {
			logg.Errorf("manager close: %v", err)
		}
	}()

	signals := make(chan model.FlexibilitySignal, 1)
	connector := rte.NewConnector(cfg.RTE, manager)
	go func() {
		if err := connector.Start(ctx); err != nil {
			logg.Errorf("connector error: %v", err)
		}
	}()
	go manager.Run(ctx, signals)

	// send an initial dummy signal so the service does some work
	signals <- model.FlexibilitySignal{Type: model.SignalFCR, Timestamp: time.Now()}

	<-ctx.Done()
	return nil
}
