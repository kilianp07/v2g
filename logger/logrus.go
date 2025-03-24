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
	}
}

func (l *logrusLogger) SetOutput(output *os.File) {
	l.logger.SetOutput(output)
}

func (l *logrusLogger) SetLevel(level Level) {
	l.logger.SetLevel(levels[level])
}

func (l *logrusLogger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *logrusLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}
