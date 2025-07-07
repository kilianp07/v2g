package test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/dispatch"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
)

type captureLogger struct {
	mu   sync.Mutex
	msgs []string
}

func (c *captureLogger) Debugf(format string, args ...any) {}
func (c *captureLogger) Infof(format string, args ...any) {
	c.mu.Lock()
	c.msgs = append(c.msgs, fmt.Sprintf(format, args...))
	c.mu.Unlock()
}
func (c *captureLogger) Warnf(format string, args ...any)  { c.Infof(format, args...) }
func (c *captureLogger) Errorf(format string, args ...any) { c.Infof(format, args...) }

func TestIntegration_SoCConstraints(t *testing.T) {
	log := &captureLogger{}
	disp := dispatch.NewSmartDispatcher()
	disp.Logger = log
	pub := mqtt.NewMockPublisher()
	mgr, err := dispatch.NewDispatchManager(dispatch.SimpleVehicleFilter{}, &disp, dispatch.NoopFallback{}, pub, time.Second, nil, nil, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	vehicles := []model.Vehicle{
		{ID: "ok", SoC: 0.8, BatteryKWh: 40, IsV2G: true, Available: true, MaxPower: 10},
		{ID: "low", SoC: 0.05, BatteryKWh: 40, IsV2G: true, Available: true, MaxPower: 10},
	}
	sig := model.FlexibilitySignal{Type: model.SignalMA, PowerKW: -5, Duration: time.Hour, Timestamp: time.Now()}
	res := mgr.Dispatch(sig, vehicles)
	if _, ok := res.Assignments["ok"]; !ok {
		t.Fatalf("expected ok vehicle assigned")
	}
	if _, ok := res.Assignments["low"]; ok {
		t.Fatalf("low SoC vehicle should be skipped")
	}
	foundSkip := false
	for _, m := range log.msgs {
		if strings.Contains(m, "skipped") && strings.Contains(m, "low") {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Fatalf("expected log for skipped vehicle, got %v", log.msgs)
	}
}
