package eco

import (
	"testing"
	"time"
)

func TestMemoryStore_Aggregation(t *testing.T) {
	s := NewMemoryStore()
	d := Day(time.Now())
	if err := s.Add(Record{VehicleID: "v1", Date: d, InjectedKWh: 2}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := s.Add(Record{VehicleID: "v1", Date: d.Add(2 * time.Hour), InjectedKWh: 1}); err != nil {
		t.Fatalf("add2: %v", err)
	}
	recs, err := s.Query("v1", d, d)
	if err != nil || len(recs) != 1 {
		t.Fatalf("query: %v len=%d", err, len(recs))
	}
	if recs[0].InjectedKWh != 3 {
		t.Fatalf("expected 3 got %f", recs[0].InjectedKWh)
	}
}

func TestRecordCalculations(t *testing.T) {
	r := Record{InjectedKWh: 4, ConsumedKWh: 2}
	if r.EnergyRatio() != 2 {
		t.Fatalf("ratio")
	}
	if r.CO2Avoided(10) != 40 {
		t.Fatalf("co2")
	}
}
