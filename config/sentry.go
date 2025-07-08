package config

// SentryConfig defines settings for Sentry error monitoring.
type SentryConfig struct {
	DSN              string  `json:"dsn"`
	Environment      string  `json:"environment"`
	TracesSampleRate float64 `json:"traces_sample_rate"`
	Release          string  `json:"release"`
}
