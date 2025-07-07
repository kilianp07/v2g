package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZerologLoggerMethods(t *testing.T) {
	assert.NoError(t, os.Setenv("APP_ENV", "dev"))
	defer assert.NoError(t, os.Unsetenv("APP_ENV"))
	l := NewZerologLogger("test")
	if l == nil {
		t.Fatalf("nil logger")
	}
	l.Debugf("debug %d", 1)
	l.Debugw("debug", map[string]any{"k": 1})
	l.Infof("info %s", "test")
	l.Warnf("warn")
	l.Errorf("error")
}
