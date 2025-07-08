package app

import (
	"context"
	"fmt"
	"time"

	"github.com/kilianp07/v2g/app/plugins"
	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/core/dispatch"
	dispatchlog "github.com/kilianp07/v2g/core/dispatch/logging"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/core/prediction"
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
	for _, exp := range cfg.Components.Metrics {
		fac, ok := plugins.MetricsExporters[exp.Type]
		if !ok {
			return nil, fmt.Errorf("unknown metrics exporter %s", exp.Type)
		}
		sink, err := fac("metrics", exp.Conf)
		if err != nil {
			return nil, fmt.Errorf("metrics exporter %s: %w", exp.Type, err)
		}
		sinks = append(sinks, sink)
	}
	if len(sinks) == 0 {
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
	dispFac, ok := plugins.Dispatchers[cfg.Components.Dispatcher.Type]
	if !ok {
		return nil, fmt.Errorf("unknown dispatcher %s", cfg.Components.Dispatcher.Type)
	}
	dispatcher, err := dispFac("dispatcher", cfg.Components.Dispatcher.Conf)
	if err != nil {
		return nil, fmt.Errorf("dispatcher %s: %w", cfg.Components.Dispatcher.Type, err)
	}

	fbFac, ok := plugins.Fallbacks[cfg.Components.Fallback.Type]
	if !ok {
		return nil, fmt.Errorf("unknown fallback %s", cfg.Components.Fallback.Type)
	}
	fallback, err := fbFac("fallback", cfg.Components.Fallback.Conf)
	if err != nil {
		return nil, fmt.Errorf("fallback %s: %w", cfg.Components.Fallback.Type, err)
	}

	tunerFac, ok := plugins.Tuners[cfg.Components.Tuner.Type]
	var tuner dispatch.LearningTuner
	if ok {
		tuner, err = tunerFac("tuner", cfg.Components.Tuner.Conf, dispatcher)
		if err != nil {
			return nil, fmt.Errorf("tuner %s: %w", cfg.Components.Tuner.Type, err)
		}
	}

	predFac, ok := plugins.Predictions[cfg.Components.Prediction.Type]
	var pred prediction.PredictionEngine
	if ok {
		pred, err = predFac("prediction", cfg.Components.Prediction.Conf)
		if err != nil {
			return nil, fmt.Errorf("prediction %s: %w", cfg.Components.Prediction.Type, err)
		}
	}

	logFac, ok := plugins.LogStores[cfg.Logging.Backend]
	var store dispatchlog.LogStore
	if ok {
		// Logging config is passed through the plugin conf mechanism
		// to allow custom fields beyond the built-in structure.
		conf := map[string]any{"path": cfg.Logging.Path, "max_size_mb": cfg.Logging.MaxSizeMB, "max_backups": cfg.Logging.MaxBackups, "max_age_days": cfg.Logging.MaxAgeDays}
		store, err = logFac("logging", conf)
		if err != nil {
			return nil, fmt.Errorf("log store: %w", err)
		}
	}

	manager, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatcher,
		fallback,
		client,
		ackTimeout,
		sink,
		bus,
		disc,
		logg,
		tuner,
		pred,
	)
	if err != nil {
		return nil, fmt.Errorf("dispatch manager: %w", err)
	}
	manager.SetLPFirst(cfg.Dispatch.LPFirst)
	if store != nil {
		manager.SetLogStore(store)
	}

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
