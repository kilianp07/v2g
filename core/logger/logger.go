package logger

// Logger exposes logging methods for common severity levels.
type Logger interface {
	Debugf(format string, args ...any)
	// Debugw logs a message with structured fields.
	Debugw(msg string, fields map[string]any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// StructuredLogger can log structured debug information. It is implemented by
// ZerologLogger and other adapters.
type StructuredLogger interface {
	Debugw(msg string, fields map[string]any)
}
