package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/kilianp07/v2g/app"
	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/infra/logger"
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
	svc, err := app.New(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := svc.Close(); err != nil {
			logger.New("main").Errorf("service close: %v", err)
		}
	}()
	return svc.Run(ctx)
}
