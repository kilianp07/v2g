package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/mqtt"
)

var fleetCmd = &cobra.Command{
	Use:   "fleet",
	Short: "Fleet related commands",
}

var fleetLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List discovered vehicles",
	RunE:  runFleetLs,
}

func init() {
	fleetCmd.AddCommand(fleetLsCmd)
	rootCmd.AddCommand(fleetCmd)
}

func runFleetLs(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	discCfg := cfg.MQTT
	suffix := time.Now().UnixNano()
	if discCfg.ClientID != "" {
		discCfg.ClientID = fmt.Sprintf("%s-%d", discCfg.ClientID, suffix)
	} else {
		discCfg.ClientID = fmt.Sprintf("fleet-ls-%d", suffix)
	}
	disc, err := mqtt.NewPahoFleetDiscovery(discCfg, "v2g/fleet/discovery", "v2g/fleet/response/+", "hello")
	if err != nil {
		return fmt.Errorf("fleet discovery: %w", err)
	}
	defer func() {
		if err := disc.Close(); err != nil {
			if _, ferr := fmt.Fprintf(cmd.ErrOrStderr(), "error while closing discovery: %v\n", err); ferr != nil {
				fmt.Println("failed to write to stderr:", ferr)
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	vehicles, err := disc.Discover(ctx, 2*time.Second)
	if err != nil {
		return err
	}
	for _, v := range vehicles {
		fmt.Println(v.ID)
	}
	return nil
}
