package plugins

import (
	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/core/dispatch"
	dispatchlog "github.com/kilianp07/v2g/core/dispatch/logging"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/prediction"
	"github.com/kilianp07/v2g/infra/logger"
	inframetrics "github.com/kilianp07/v2g/infra/metrics"
	"github.com/mitchellh/mapstructure"
)

func init() {
	RegisterDispatcher("equal", func(name string, _ map[string]any) (dispatch.Dispatcher, error) {
		return dispatch.EqualDispatcher{}, nil
	})
	RegisterDispatcher("smart", func(name string, _ map[string]any) (dispatch.Dispatcher, error) {
		d := dispatch.NewSmartDispatcher()
		return &d, nil
	})
	RegisterDispatcher("lp", func(name string, _ map[string]any) (dispatch.Dispatcher, error) {
		lp := dispatch.NewLPDispatcher()
		return &lp, nil
	})
	RegisterDispatcher("segmented", func(name string, _ map[string]any) (dispatch.Dispatcher, error) {
		return dispatch.NewSegmentedSmartDispatcher(nil), nil
	})

	RegisterFallback("noop", func(name string, _ map[string]any) (dispatch.FallbackStrategy, error) {
		return dispatch.NoopFallback{}, nil
	})
	RegisterFallback("balanced", func(name string, _ map[string]any) (dispatch.FallbackStrategy, error) {
		return dispatch.NewBalancedFallback(logger.New("fallback")), nil
	})
	RegisterFallback("probabilistic", func(name string, _ map[string]any) (dispatch.FallbackStrategy, error) {
		return dispatch.NewProbabilisticFallback(logger.New("fallback")), nil
	})

	RegisterTuner("ack", func(name string, _ map[string]any, d dispatch.Dispatcher) (dispatch.LearningTuner, error) {
		sd, ok := d.(*dispatch.SmartDispatcher)
		if !ok {
			return nil, nil
		}
		return dispatch.NewAckBasedTuner(sd), nil
	})

	RegisterMetrics("prometheus", func(name string, conf map[string]any) (coremetrics.MetricsSink, error) {
		var mc coremetrics.Config
		if err := mapstructure.Decode(conf, &mc); err != nil {
			return nil, err
		}
		return inframetrics.NewPromSink(mc)
	})
	RegisterMetrics("influx", func(name string, conf map[string]any) (coremetrics.MetricsSink, error) {
		var mc coremetrics.Config
		if err := mapstructure.Decode(conf, &mc); err != nil {
			return nil, err
		}
		return inframetrics.NewInfluxSinkWithFallback(mc), nil
	})

	RegisterLogStore("jsonl", func(name string, conf map[string]any) (dispatchlog.LogStore, error) {
		var lc config.LoggingConfig
		if err := mapstructure.Decode(conf, &lc); err != nil {
			return nil, err
		}
		return dispatchlog.NewJSONLStore(lc.Path)
	})
	RegisterLogStore("sqlite", func(name string, conf map[string]any) (dispatchlog.LogStore, error) {
		var lc config.LoggingConfig
		if err := mapstructure.Decode(conf, &lc); err != nil {
			return nil, err
		}
		return dispatchlog.NewSQLiteStore(lc.Path)
	})

	RegisterPrediction("mock", func(name string, _ map[string]any) (prediction.PredictionEngine, error) {
		return &prediction.MockPredictionEngine{}, nil
	})
}
