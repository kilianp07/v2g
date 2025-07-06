package dispatch

import (
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
	vehiclestatus "github.com/kilianp07/v2g/core/vehiclestatus"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/mqtt"
)

type fakeStatusStore struct {
	calls map[string]vehiclestatus.LastDispatch
}

func (f *fakeStatusStore) Set(vehiclestatus.Status)                         {}
func (f *fakeStatusStore) List(vehiclestatus.Filter) []vehiclestatus.Status { return nil }
func (f *fakeStatusStore) RecordDispatch(id string, dec vehiclestatus.LastDispatch) {
	if f.calls == nil {
		f.calls = make(map[string]vehiclestatus.LastDispatch)
	}
	f.calls[id] = dec
}

func TestDispatchManager_RecordDispatch(t *testing.T) {
	store := &fakeStatusStore{}
	pub := mqtt.NewMockPublisher()
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, pub, time.Second, nil, nil, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	mgr.SetStatusStore(store)
	vehicles := []model.Vehicle{
		{ID: "v1", IsV2G: true, Available: true, MaxPower: 10, SoC: 0.8},
		{ID: "v2", IsV2G: true, Available: true, MaxPower: 10, SoC: 0.8},
	}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Timestamp: time.Now()}
	mgr.Dispatch(sig, vehicles)
	if len(store.calls) != 2 {
		t.Fatalf("expected 2 calls got %d", len(store.calls))
	}
	for _, id := range []string{"v1", "v2"} {
		dec, ok := store.calls[id]
		if !ok {
			t.Fatalf("missing call for %s", id)
		}
		if dec.SignalType != "FCR" || dec.TargetPower != 5 {
			t.Errorf("wrong dispatch data for %s: %#v", id, dec)
		}
	}
}

func TestDispatchManager_NoStatusStore(t *testing.T) {
	store := &fakeStatusStore{}
	pub := mqtt.NewMockPublisher()
	mgr, err := NewDispatchManager(SimpleVehicleFilter{}, EqualDispatcher{}, NoopFallback{}, pub, time.Second, nil, nil, nil, logger.NopLogger{}, nil, nil)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	mgr.SetStatusStore(nil)
	vehicles := []model.Vehicle{{ID: "v1", IsV2G: true, Available: true, MaxPower: 10, SoC: 0.8}}
	sig := model.FlexibilitySignal{Type: model.SignalFCR, PowerKW: 5, Timestamp: time.Now()}
	mgr.Dispatch(sig, vehicles)
	if len(store.calls) != 0 {
		t.Fatalf("expected 0 calls got %d", len(store.calls))
	}
}
