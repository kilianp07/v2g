package test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/core/scheduler"
	"github.com/kilianp07/v2g/pkg/export"
)

func TestSchedulerIntegration(t *testing.T) {
	date := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	cfg := scheduler.SchedulerConfig{SlotDurationMinutes: 60, TargetEnergyKWh: 4}
	vehicles := []model.Vehicle{{ID: "v1", MaxPower: 2}, {ID: "v2", MaxPower: 2}}
	avail := map[string][]scheduler.AvailabilityWindow{
		"v1": {{Start: date, End: date.Add(24 * time.Hour)}},
		"v2": {{Start: date, End: date.Add(24 * time.Hour)}},
	}
	s := scheduler.Scheduler{Config: cfg, Vehicles: vehicles, Availability: avail}
	plan, err := s.GeneratePlan(date)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	// verify energy distribution
	slotDur := time.Duration(cfg.SlotDurationMinutes) * time.Minute
	total := 0.0
	for _, e := range plan {
		total += e.PowerKW * slotDur.Hours()
	}
	if total < cfg.TargetEnergyKWh-1e-6 || total > cfg.TargetEnergyKWh+1e-6 {
		t.Fatalf("energy mismatch %.3f", total)
	}
	// JSON export round trip
	var buf bytes.Buffer
	if err := export.WriteJSON(&buf, plan); err != nil {
		t.Fatalf("json: %v", err)
	}
	var back []scheduler.EffacementEntry
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back) != len(plan) {
		t.Fatalf("json size mismatch")
	}
	// CSV export parse
	buf.Reset()
	if err := export.WriteCSV(&buf, plan); err != nil {
		t.Fatalf("csv: %v", err)
	}
	r := csv.NewReader(&buf)
	recs, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv read: %v", err)
	}
	if len(recs) != len(plan)+1 {
		t.Fatalf("csv rows %d", len(recs))
	}
	if recs[0][0] != "vehicle_id" {
		t.Fatalf("csv header")
	}
}
