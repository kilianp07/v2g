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
	"github.com/kilianp07/v2g/logger"
	"github.com/kilianp07/v2g/model"
	"github.com/kilianp07/v2g/mqtt"
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

	logg := logger.New("dispatch")
	mqttCfg := cfg.MQTT
	mqttCfg.Logger = logg
	client, err := mqtt.NewPahoClient(mqttCfg)
	if err != nil {
		return fmt.Errorf("mqtt client: %w", err)
	}

	manager, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatch.EqualDispatcher{},
		dispatch.NoopFallback{},
		client,
		time.Duration(cfg.Dispatch.AckTimeoutSeconds)*time.Second,
		logg,
		nil,
	)
	if err != nil {
		return fmt.Errorf("dispatch manager: %w", err)
	}

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
