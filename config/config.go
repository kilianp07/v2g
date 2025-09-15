package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/kilianp07/v2g/core/dispatch"
	"github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/infra/mqtt"
)

type Config struct {
	MQTT         mqtt.Config        `json:"mqtt"`
	Dispatch     dispatch.Config    `json:"dispatch"`
	Metrics      metrics.Config     `json:"metrics"`
	Logging      LoggingConfig      `json:"logging"`
	RTE          RTEConfig          `json:"rte"`
	RTEGenerator RTEGeneratorConfig `json:"rteGenerator"`
	Sentry       SentryConfig       `json:"sentry"`
	Telemetry    TelemetryConfig    `json:"telemetry"`
}

func Load(path string) (*Config, error) {
	k := koanf.New(".")
	ext := strings.ToLower(filepath.Ext(path))
	var parser koanf.Parser
	switch ext {
	case ".yaml", ".yml":
		parser = yaml.Parser()
	case ".json":
		parser = json.Parser()
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}
	if err := k.Load(file.Provider(path), parser); err != nil {
		return nil, err
	}
	// Optional environment overrides
	if err := k.Load(env.Provider("K_", "__", func(s string) string {
		s = strings.TrimPrefix(strings.ToLower(s), "k_")
		return strings.ReplaceAll(s, "__", ".")
	}), nil); err != nil {
		return nil, err
	}
	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		return nil, err
	}
	cfg.Logging.SetDefaults()
	cfg.RTEGenerator.SetDefaults()
	if err := cfg.RTE.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.RTEGenerator.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.Logging.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}
