package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// ZerologLogger implements Logger using rs/zerolog.
type ZerologLogger struct {
	log zerolog.Logger
}

// NewZerologLogger creates a ZerologLogger using the APP_ENV environment variable
// to determine the output format. All logs include the provided component field.
func NewZerologLogger(component string) Logger {
	env := strings.ToLower(os.Getenv("APP_ENV"))
	var z zerolog.Logger
	if env == "dev" {
		writer := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		z = zerolog.New(writer).With().Timestamp().Str("component", component).Logger()
	} else {
		z = zerolog.New(os.Stdout).With().Timestamp().Str("component", component).Logger()
	}
	return &ZerologLogger{log: z}
}

func (l *ZerologLogger) Debugf(format string, args ...any) {
	l.log.Debug().Msgf(format, args...)
}

func (l *ZerologLogger) Debugw(msg string, fields map[string]any) {
	ev := l.log.Debug()
	for k, v := range fields {
		ev = ev.Interface(k, v)
	}
	ev.Msg(msg)
}

func (l *ZerologLogger) Infof(format string, args ...any) {
	l.log.Info().Msgf(format, args...)
}

func (l *ZerologLogger) Warnf(format string, args ...any) {
	l.log.Warn().Msgf(format, args...)
}

func (l *ZerologLogger) Errorf(format string, args ...any) {
	l.log.Error().Msgf(format, args...)
}
