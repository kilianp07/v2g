package dispatch

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
	coremon "github.com/kilianp07/v2g/core/monitoring"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
)

type recordMonitor struct {
	err  error
	tags map[string]string
}

func (r *recordMonitor) CaptureException(err error, tags map[string]string) {
	r.err = err
	r.tags = tags
}
func (r *recordMonitor) Recover()            {}
func (r *recordMonitor) Flush(time.Duration) {}

func TestDispatchErrorCaptured(t *testing.T) {
	pub := mqtt.NewMockPublisher()
	pub.FailIDs["v1"] = true
	mon := &recordMonitor{}
	coremon.Init(mon)
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, pub, time.Second, nil, nil, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	vehicles := []model.Vehicle{{ID: "v1", IsV2G: true, Available: true, MaxPower: 10, SoC: 0.8}}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Timestamp: time.Now()}
	mgr.Dispatch(sig, vehicles)
	if mon.err == nil {
		t.Fatalf("error not captured")
	}
	if mon.tags["vehicle_id"] != "v1" || mon.tags["module"] != "dispatch_manager" {
		t.Fatalf("tags missing")
	}
	coremon.Init(coremon.NopMonitor{})
}
