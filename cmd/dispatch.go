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
	"github.com/kilianp07/v2g/core/dispatch"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
)

var dispatchCmd = &cobra.Command{
	Use:   "dispatch",
	Short: "Inject a test flexibility signal",
	RunE:  dispatchSignal,
}

func init() {
	rootCmd.AddCommand(dispatchCmd)
}

func dispatchSignal(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logg := logger.New("dispatch-command")
	mqttCfg := cfg.MQTT
	client, err := mqtt.NewPahoClient(mqttCfg)
	if err != nil {
		return fmt.Errorf("mqtt client: %w", err)
	}

	bus := eventbus.New()
	disc, err := mqtt.NewPahoFleetDiscovery(mqttCfg, "v2g/fleet/discovery", "v2g/fleet/response/+", "hello")
	if err != nil {
		return fmt.Errorf("fleet discovery: %w", err)
	}
	manager, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatch.EqualDispatcher{},
		dispatch.NoopFallback{},
		client,
		time.Duration(cfg.Dispatch.AckTimeoutSeconds)*time.Second,
		nil,
		bus,
		disc,
		logg,
		nil,
	)
	if err != nil {
		return fmt.Errorf("dispatch manager: %w", err)
	}
	manager.SetLPFirst(cfg.Dispatch.LPFirst)
	defer func() {
		if err := manager.Close(); err != nil {
			logg.Errorf("manager close: %v", err)
		}
	}()

	veh := model.Vehicle{ID: "test", IsV2G: true, Available: true, MaxPower: 10, BatteryKWh: 40, SoC: 0.8}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Duration: time.Minute, Timestamp: time.Now()}
	res := manager.Dispatch(sig, []model.Vehicle{veh})
	if len(res.Errors) > 0 {
		for id, derr := range res.Errors {
			logg.Errorf("dispatch %s failed: %v", id, derr)
		}
		return fmt.Errorf("dispatch encountered errors")
	}

	<-ctx.Done()
	return nil
}
