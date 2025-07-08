package monitoring

import (
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/kilianp07/v2g/config"
	coremon "github.com/kilianp07/v2g/core/monitoring"
)

// NewSentryMonitor initializes Sentry using the provided configuration and
// returns a Monitor implementation.
func NewSentryMonitor(cfg config.SentryConfig) (coremon.Monitor, error) {
	if cfg.DSN == "" {
		return coremon.NopMonitor{}, nil
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		TracesSampleRate: cfg.TracesSampleRate,
		Release:          cfg.Release,
	})
	if err != nil {
		return nil, err
	}
	return &sentryMonitor{}, nil
}

type sentryMonitor struct{}

func (s *sentryMonitor) CaptureException(err error, tags map[string]string) {
	if err == nil {
		return
	}
	if len(tags) == 0 {
		sentry.CaptureException(err)
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		sentry.CaptureException(err)
	})
}

func (s *sentryMonitor) Recover() {
	if r := recover(); r != nil {
		sentry.CurrentHub().Recover(r)
		sentry.Flush(2 * time.Second)
		panic(r)
	}
}

func (s *sentryMonitor) Flush(timeout time.Duration) { sentry.Flush(timeout) }
