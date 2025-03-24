package formatter

import (
	"time"

	"github.com/sirupsen/logrus"
)

type LogrusFormatterAdapter struct {
	Formatter logrus.Formatter
}

func (f *LogrusFormatterAdapter) Format(data map[string]any) ([]byte, error) {
	entry := &logrus.Entry{
		Data: data,
	}

	if message, ok := data["message"].(string); ok {
		entry.Message = message
	}
	if level, ok := data["level"].(string); ok {
		entry.Level, _ = logrus.ParseLevel(level)
	}
	if timeStr, ok := data["time"].(string); ok {
		parsedTime, err := time.Parse(time.RFC3339, timeStr)
		if err == nil {
			entry.Time = parsedTime
		}
	}

	return f.Formatter.Format(entry)
}
