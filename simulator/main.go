package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	cfg := parseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	strat := RandomAck{Delay: cfg.AckLatency, DropRate: cfg.DropRate}

	var wg sync.WaitGroup
	for i := 0; i < cfg.Count; i++ {
		id := fmt.Sprintf("veh%03d", i+1)
		v := NewSimulatedVehicle(id, cfg.Broker, strat)
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

func parseFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.Broker, "broker", "tcp://localhost:1883", "MQTT broker URL")
	flag.IntVar(&cfg.Count, "count", 1, "number of vehicles")
	flag.DurationVar(&cfg.AckLatency, "ack-latency", 0, "ack latency")
	flag.Float64Var(&cfg.DropRate, "drop-rate", 0, "ack drop rate")
	flag.Parse()
	return cfg
}
