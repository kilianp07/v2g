package logger

// Logger exposes logging methods for common severity levels.
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// NopLogger implements Logger with no-op methods.
type NopLogger struct{}

func (NopLogger) Debugf(string, ...any) {}
func (NopLogger) Infof(string, ...any)  {}
func (NopLogger) Warnf(string, ...any)  {}
func (NopLogger) Errorf(string, ...any) {}

// New returns a Logger for the given component. The environment is detected via
// the APP_ENV variable.
func New(component string) Logger {
	return NewZerologLogger(component)
}
