package logger

import corelogger "github.com/kilianp07/v2g/core/logger"

// Alias the core interface for convenience.
// Logger mirrors the core logger interface.
type Logger = corelogger.Logger

// NopLogger implements Logger with no-op methods.
type NopLogger struct{}

func (NopLogger) Debugf(string, ...any)         {}
func (NopLogger) Debugw(string, map[string]any) {}
func (NopLogger) Infof(string, ...any)          {}
func (NopLogger) Warnf(string, ...any)          {}
func (NopLogger) Errorf(string, ...any)         {}

// New returns a Logger for the given component. The environment is detected via
// the APP_ENV variable.
func New(component string) Logger {
	return NewZerologLogger(component)
}
