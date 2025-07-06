package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/infra/metrics"
)

func main() {
	cfg := parseFlags()
	if err := (&cfg).Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	if !cfg.Verbose {
		log.SetOutput(io.Discard)
	}

	applyBatteryProfile(&cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	strat := RandomAck{Delay: cfg.AckLatency, DropRate: cfg.DropRate}
	var sink coremetrics.MetricsSink = coremetrics.NopSink{}
	if cfg.InfluxURL != "" {
		sink = metrics.NewInfluxSinkWithFallback(coremetrics.Config{
			InfluxEnabled: true,
			InfluxURL:     cfg.InfluxURL,
			InfluxToken:   cfg.InfluxToken,
			InfluxOrg:     cfg.InfluxOrg,
			InfluxBucket:  cfg.InfluxBucket,
		})
	}

	var tmpl map[string]VehicleTemplate
	var err error
	if cfg.TemplateFile != "" {
		tmpl, err = readTemplateFile(cfg.TemplateFile)
		if err != nil {
			log.Fatalf("template file: %v", err)
		}
	}

	var prof [24]float64
	if cfg.AvailabilityFile != "" {
		prof, err = readAvailabilityFile(cfg.AvailabilityFile)
		if err != nil {
			log.Fatalf("availability file: %v", err)
		}
	} else {
		for i := range prof {
			prof[i] = 1
		}
	}

	schedule := map[string]time.Time{}
	if cfg.ScheduleFile != "" {
		schedule, err = readScheduleFile(cfg.ScheduleFile)
		if err != nil {
			log.Fatalf("schedule file: %v", err)
		}
	}
	fleetCfg := FleetConfig{
		Size:           cfg.FleetSize,
		CommuterPct:    cfg.CommuterPct,
		DisconnectRate: cfg.DisconnectRate,
		Availability:   prof,
		Schedule:       schedule,
	}
	vehicles := GenerateFleet(fleetCfg, tmpl)
	runVehicles(ctx, vehicles, cfg, strat, sink)
}

func parseFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.Broker, "broker", "tcp://localhost:1883", "MQTT broker URL")
	flag.IntVar(&cfg.Count, "count", 1, "number of vehicles")
	flag.IntVar(&cfg.FleetSize, "fleet-size", 0, "auto generated fleet size")
	flag.DurationVar(&cfg.AckLatency, "ack-latency", 0, "ack latency")
	flag.Float64Var(&cfg.DropRate, "drop-rate", 0, "ack drop rate")
	flag.Float64Var(&cfg.DisconnectRate, "disconnect-rate", 0, "disconnect probability per minute")
	flag.Float64Var(&cfg.CapacityKWh, "capacity", 40, "battery capacity kWh")
	flag.Float64Var(&cfg.ChargeRateKW, "charge-rate", 7, "charge rate kW")
	flag.Float64Var(&cfg.DischargeRateKW, "discharge-rate", 10, "discharge rate kW")
	flag.Float64Var(&cfg.MaxPower, "max-power", 10, "vehicle max power kW")
	flag.DurationVar(&cfg.Interval, "interval", time.Second*30, "SoC publish interval")
	flag.Float64Var(&cfg.CommuterPct, "commuter-pct", 0, "ratio of commuter vehicles")
	flag.StringVar(&cfg.AvailabilityFile, "availability-file", "", "hourly availability JSON")
	flag.StringVar(&cfg.ScheduleFile, "schedule-file", "", "schedule overrides file")
	flag.StringVar(&cfg.TemplateFile, "template-file", "", "vehicle template overrides")
	flag.StringVar(&cfg.BatteryProfile, "battery-profile", "", "predefined battery profile (small,medium,large)")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "enable verbose logging")
	flag.StringVar(&cfg.TopicPrefix, "topic-prefix", "v2g", "MQTT topic prefix")
	flag.StringVar(&cfg.InfluxURL, "influx-url", "", "InfluxDB URL")
	flag.StringVar(&cfg.InfluxToken, "influx-token", "", "InfluxDB token")
	flag.StringVar(&cfg.InfluxOrg, "influx-org", "", "InfluxDB organization")
	flag.StringVar(&cfg.InfluxBucket, "influx-bucket", "", "InfluxDB bucket")
	flag.Parse()
	return cfg
}

func readTemplateFile(path string) (map[string]VehicleTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]VehicleTemplate
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func readAvailabilityFile(path string) ([24]float64, error) {
	var prof [24]float64
	data, err := os.ReadFile(path)
	if err != nil {
		return prof, err
	}
	return LoadAvailabilityProfile(data)
}

func readScheduleFile(path string) (map[string]time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	sched := make(map[string]time.Time, len(raw))
	for id, ts := range raw {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("invalid departure for %s: %w", id, err)
		}
		sched[id] = t
	}
	return sched, nil
}

func applyBatteryProfile(cfg *Config) {
	switch cfg.BatteryProfile {
	case "small":
		cfg.CapacityKWh = 20
		cfg.ChargeRateKW = 3.6
		cfg.DischargeRateKW = 7
	case "medium":
		cfg.CapacityKWh = 40
		cfg.ChargeRateKW = 7
		cfg.DischargeRateKW = 10
	case "large":
		cfg.CapacityKWh = 80
		cfg.ChargeRateKW = 11
		cfg.DischargeRateKW = 20
	case "":
	default:
		log.Printf("unknown battery profile %s", cfg.BatteryProfile)
	}
}

func runVehicles(ctx context.Context, vehicles []SimulatedVehicle, cfg Config, strat AckStrategy, sink coremetrics.MetricsSink) {
	var wg sync.WaitGroup
	for i := range vehicles {
		b := &Battery{
			CapacityKWh:     cfg.CapacityKWh,
			Soc:             0.8,
			ChargeRateKW:    cfg.ChargeRateKW,
			DischargeRateKW: cfg.DischargeRateKW,
		}
		v := &vehicles[i]
		v.Broker = cfg.Broker
		v.TopicPrefix = cfg.TopicPrefix
		v.Strategy = strat
		v.Interval = cfg.Interval
		v.MaxPower = cfg.MaxPower
		v.Battery = b
		v.Metrics = sink
		wg.Add(1)
		go func(v *SimulatedVehicle) {
			defer wg.Done()
			if err := v.Run(ctx); err != nil {
				log.Printf("%s: %v", v.ID, err)
			}
		}(v)
	}
	wg.Wait()
}
