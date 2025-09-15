package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	coremetrics "github.com/kilianp07/v2g/core/metrics"
	coremodel "github.com/kilianp07/v2g/core/model"
	coremon "github.com/kilianp07/v2g/core/monitoring"
	inframetrics "github.com/kilianp07/v2g/infra/metrics"
	inframon "github.com/kilianp07/v2g/infra/monitoring"

	"github.com/kilianp07/v2g/app"
	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/infra/logger"
)

var cfgPath string
var demoSeed bool
var rteGen bool
var rteGenScenario string

var rootCmd = &cobra.Command{
	Use:   "v2g",
	Short: "V2G dispatch service",
	RunE:  run,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "config.yaml", "configuration file")
	rootCmd.PersistentFlags().BoolVar(&demoSeed, "demo-seed", false, "write sample metrics to Influx and exit")
	rootCmd.PersistentFlags().BoolVar(&rteGen, "rte-gen", false, "force enable RTE generator")
	rootCmd.PersistentFlags().StringVar(&rteGenScenario, "rte-gen-scenario", "", "override RTE generator scenario")
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
	if rteGen {
		cfg.RTEGenerator.Enabled = true
	}
	if rteGenScenario != "" {
		cfg.RTEGenerator.Scenario = rteGenScenario
	}
	mon, err := inframon.NewSentryMonitor(cfg.Sentry)
	if err != nil {
		logger.New("main").Warnf("Sentry init failed: %v", err)
	}
	coremon.Init(mon)
	defer coremon.Flush(2 * time.Second)
	svc, err := app.New(cfg)
	if err != nil {
		return err
	}
	if demoSeed && cfg.Metrics.InfluxEnabled {
		if err := seedDemo(cfg.Metrics); err != nil {
			logger.New("main").Errorf("demo seed: %v", err)
		}
		return nil
	}
	defer func() {
		if err := svc.Close(); err != nil {
			logger.New("main").Errorf("service close: %v", err)
		}
	}()
	return svc.Run(ctx)
}

func seedDemo(cfg coremetrics.Config) error {
	sink := inframetrics.NewInfluxSink(cfg.InfluxURL, cfg.InfluxToken, cfg.InfluxOrg, cfg.InfluxBucket)
	now := time.Now()
	v := coremodel.Vehicle{ID: "demo-veh", SoC: 0.5, Available: true, MaxPower: 7}
	_ = sink.RecordVehicleState(coremetrics.VehicleStateEvent{Vehicle: v, Time: now})
	_ = sink.RecordDispatchOrder(coremetrics.DispatchOrderEvent{OrderID: "demo1", VehicleID: v.ID, Signal: coremodel.SignalFCR, PowerKW: 3.3, Score: 0.9, Accepted: true, Time: now})
	_ = sink.RecordDispatchAck(coremetrics.DispatchAckEvent{OrderID: "demo1", VehicleID: v.ID, Signal: coremodel.SignalFCR, Acknowledged: true, Latency: 50 * time.Millisecond, Time: now.Add(50 * time.Millisecond)})
	_ = sink.RecordRTESignal(coremetrics.RTESignalEvent{Signal: coremodel.FlexibilitySignal{Type: coremodel.SignalFCR, PowerKW: 5, Duration: time.Minute}, Time: now})
	return nil
}
