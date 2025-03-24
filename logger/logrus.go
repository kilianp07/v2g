package logger

import (
	"os"

	"github.com/kilianp07/v2g/logger/formatter"
	"github.com/sirupsen/logrus"
)

var levels = map[Level]logrus.Level{
	DebugLevel: logrus.DebugLevel,
	InfoLevel:  logrus.InfoLevel,
	WarnLevel:  logrus.WarnLevel,
	ErrorLevel: logrus.ErrorLevel,
	FatalLevel: logrus.FatalLevel,
}

type logrusLogger struct {
	logger *logrus.Logger
}

func (l *logrusLogger) SetFormatter(f formatter.Formatter) {
	if adapter, ok := f.(*formatter.LogrusFormatterAdapter); ok {
		l.logger.SetFormatter(adapter.Formatter)
	} else {
		l.logger.Warn("Attempted to set incompatible formatter type. Expected *formatter.LogrusFormatterAdapter")
	}
}

func (l *logrusLogger) SetOutput(output *os.File) {
	l.logger.SetOutput(output)
}

func (l *logrusLogger) SetLevel(level Level) {
	l.logger.SetLevel(levels[level])
}

func (l *logrusLogger) Debug(args ...any) {
	l.logger.Debug(args...)
}

func (l *logrusLogger) Info(args ...any) {
	l.logger.Info(args...)
}

func (l *logrusLogger) Warn(args ...any) {
	l.logger.Warn(args...)
}

func (l *logrusLogger) Error(args ...any) {
	l.logger.Error(args...)
}

func (l *logrusLogger) Fatal(args ...any) {
	l.logger.Fatal(args...)
}

func (l *logrusLogger) WithField(key string, value any) Logger {
	entry := l.logger.WithField(key, value)
	return &logrusEntryLogger{entry: entry}
}

func (l *logrusLogger) WithFields(fields map[string]any) Logger {
	entry := l.logger.WithFields(logrus.Fields(fields))
	return &logrusEntryLogger{entry: entry}
}

type logrusEntryLogger struct {
	entry *logrus.Entry
}

func (e *logrusEntryLogger) SetFormatter(formatter formatter.Formatter) {
	// No-op for entry
}

func (e *logrusEntryLogger) SetOutput(output *os.File) {
	// No-op for entry
}

func (e *logrusEntryLogger) SetLevel(level Level) {
	// No-op for entry
}

func (e *logrusEntryLogger) Debug(args ...any) {
	e.entry.Debug(args...)
}

func (e *logrusEntryLogger) Info(args ...any) {
	e.entry.Info(args...)
}

func (e *logrusEntryLogger) Warn(args ...any) {
	e.entry.Warn(args...)
}

func (e *logrusEntryLogger) Error(args ...any) {
	e.entry.Error(args...)
}

func (e *logrusEntryLogger) Fatal(args ...any) {
	e.entry.Fatal(args...)
}

func (e *logrusEntryLogger) WithField(key string, value any) Logger {
	return &logrusEntryLogger{entry: e.entry.WithField(key, value)}
}

func (e *logrusEntryLogger) WithFields(fields map[string]any) Logger {
	return &logrusEntryLogger{entry: e.entry.WithFields(logrus.Fields(fields))}
}
