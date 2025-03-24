package logger

import (
	"os"

	"github.com/kilianp07/v2g/logger/formatter"
	"github.com/sirupsen/logrus"
)

type Logger interface {
	SetFormatter(formatter formatter.Formatter)
	SetOutput(output *os.File)
	SetLevel(level Level)
	Info(args ...any)
	Error(args ...any)
}

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var Log Logger = &logrusLogger{
	logger: logrus.New(),
}

func init() {
	Log.SetFormatter(&formatter.LogrusFormatterAdapter{Formatter: &logrus.JSONFormatter{}})
	Log.SetOutput(os.Stdout)
	Log.SetLevel(InfoLevel)
}
