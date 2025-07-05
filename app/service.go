package app

import (
	"context"
	"fmt"
	"time"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/core/dispatch"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/metrics"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
	"github.com/kilianp07/v2g/rte"
)

// Service orchestrates the dispatch manager and connectors.
type Service struct {
	Manager     *dispatch.DispatchManager
	Connector   rte.RTEConnector
	bus         eventbus.EventBus
	log         logger.Logger
	promEnabled bool
	promPort    string
}

// New creates a Service from the configuration.
func New(cfg *config.Config) (*Service, error) {
	logg := logger.New("service")
	client, err := mqtt.NewPahoClient(cfg.MQTT)
	if err != nil {
		return nil, fmt.Errorf("mqtt client: %w", err)
	}

	var sinks []coremetrics.MetricsSink
	promEnabled := cfg.Metrics.PrometheusEnabled
	promPort := cfg.Metrics.PrometheusPort
	if promEnabled {
		sink, err := metrics.NewPromSink(cfg.Metrics)
		if err != nil {
			return nil, fmt.Errorf("prom sink: %w", err)
		}
		sinks = append(sinks, sink)
	}
	if cfg.Metrics.InfluxEnabled {
		sink := metrics.NewInfluxSinkWithFallback(cfg.Metrics)
		sinks = append(sinks, sink)
	}
	var sink coremetrics.MetricsSink
	if len(sinks) == 1 {
		sink = sinks[0]
	} else if len(sinks) > 1 {
		sink = metrics.NewMultiSink(sinks...)
	}

	bus := eventbus.New()
	disc, err := mqtt.NewPahoFleetDiscovery(cfg.MQTT, "v2g/fleet/discovery", "v2g/fleet/response/+", "hello")
	if err != nil {
		return nil, fmt.Errorf("fleet discovery: %w", err)
	}
	ackTimeout := time.Duration(cfg.Dispatch.AckTimeoutSeconds) * time.Second
	manager, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatch.EqualDispatcher{},
		dispatch.NoopFallback{},
		client,
		ackTimeout,
		sink,
		bus,
		disc,
		logg,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("dispatch manager: %w", err)
	}
	manager.SetLPFirst(cfg.Dispatch.LPFirst)

	svc := &Service{Manager: manager, bus: bus, log: logg, promEnabled: promEnabled, promPort: promPort}
	svc.Connector = rte.NewConnector(cfg.RTE, manager)
	return svc, nil
}

// Run starts the service and blocks until the context is cancelled.
func (s *Service) Run(ctx context.Context) error {
	signals := make(chan model.FlexibilitySignal, 1)
	go s.Manager.Run(ctx, signals)
	go func() {
		if err := s.Connector.Start(ctx); err != nil {
			s.log.Errorf("connector error: %v", err)
		}
	}()
	if s.promEnabled {
		go func() {
			if err := metrics.StartPromServer(ctx, s.promPort); err != nil {
				s.log.Errorf("prom server: %v", err)
			}
		}()
	}
	signals <- model.FlexibilitySignal{Type: model.SignalFCR, Timestamp: time.Now()}
	<-ctx.Done()
	return nil
}

// Close releases resources held by the service.
func (s *Service) Close() error { return s.Manager.Close() }
