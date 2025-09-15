package config

// TelemetryConfig holds configuration for the telemetry manager.
type TelemetryConfig struct {
	Enabled         bool   `json:"enabled"`
	Mode            string `json:"mode"`
	IntervalSeconds int    `json:"interval_seconds"`
	RequestTopic    string `json:"request_topic"`
	ResponsePrefix  string `json:"response_topic_prefix"`
	StatePrefix     string `json:"state_topic_prefix"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

func (c TelemetryConfig) Interval() int {
	if c.IntervalSeconds <= 0 {
		return 10
	}
	return c.IntervalSeconds
}

func (c TelemetryConfig) Timeout() int {
	if c.TimeoutSeconds <= 0 {
		return 3
	}
	return c.TimeoutSeconds
}
