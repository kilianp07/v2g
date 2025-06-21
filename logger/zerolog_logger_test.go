package logger

import (
	"os"
	"testing"
)

func TestZerologLoggerMethods(t *testing.T) {
	os.Setenv("APP_ENV", "dev")
	defer os.Unsetenv("APP_ENV")
	l := NewZerologLogger("test")
	if l == nil {
		t.Fatalf("nil logger")
	}
	l.Debugf("debug %d", 1)
	l.Infof("info %s", "test")
	l.Warnf("warn")
	l.Errorf("error")
}
