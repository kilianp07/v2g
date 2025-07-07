package config

import (
	"fmt"
)

// LoggingConfig defines settings for dispatch log storage and rotation.
type LoggingConfig struct {
	// Backend selects the log store type: "jsonl" or "sqlite".
	Backend string `json:"backend"`
	// Path is the file location of the log store.
	Path string `json:"path"`
	// MaxSizeMB triggers rotation when the file exceeds this size in megabytes.
	MaxSizeMB int `json:"max_size_mb"`
	// MaxBackups limits the number of rotated files to keep.
	MaxBackups int `json:"max_backups"`
	// MaxAgeDays removes rotated files older than this number of days.
	MaxAgeDays int `json:"max_age_days"`
}

// SetDefaults applies sane defaults.
func (c *LoggingConfig) SetDefaults() {
	if c.Backend == "" {
		c.Backend = "jsonl"
	}
	if c.Path == "" {
		c.Path = "dispatch.log"
	}
}

// Validate checks mandatory fields.
func (c LoggingConfig) Validate() error {
	if c.Backend != "jsonl" && c.Backend != "sqlite" {
		return fmt.Errorf("unknown backend %s", c.Backend)
	}
	if c.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}
