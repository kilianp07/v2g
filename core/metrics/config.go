package metrics

// Config defines settings for metrics sinks.
type Config struct {
	PrometheusEnabled bool   `json:"prometheus_enabled"`
	PrometheusPort    string `json:"prometheus_port"`
	InfluxEnabled     bool   `json:"influx_enabled"`
	InfluxURL         string `json:"influx_url"`
	InfluxToken       string `json:"influx_token"`
	InfluxOrg         string `json:"influx_org"`
	InfluxBucket      string `json:"influx_bucket"`

	EmissionFactor float64 `json:"emission_factor"`
}
