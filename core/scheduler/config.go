package scheduler

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads SchedulerConfig from a JSON or YAML file.
func LoadConfig(path string) (SchedulerConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return SchedulerConfig{}, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var cfg SchedulerConfig
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(b, &cfg)
	case ".json":
		err = json.Unmarshal(b, &cfg)
	default:
		return SchedulerConfig{}, fmt.Errorf("unsupported config format: %s", ext)
	}
	return cfg, err
}

// DecodeConfig reads from r to decode a SchedulerConfig.
func DecodeConfig(r io.Reader, format string) (SchedulerConfig, error) {
	var cfg SchedulerConfig
	switch strings.ToLower(format) {
	case "yaml", "yml":
		dec := yaml.NewDecoder(r)
		if err := dec.Decode(&cfg); err != nil {
			return cfg, err
		}
	case "json":
		dec := json.NewDecoder(r)
		if err := dec.Decode(&cfg); err != nil {
			return cfg, err
		}
	default:
		return cfg, fmt.Errorf("unsupported format: %s", format)
	}
	return cfg, nil
}
