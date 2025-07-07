package scheduler

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

func TestGeneratePlanTotalEnergy(t *testing.T) {
	cfg := SchedulerConfig{SlotDurationMinutes: 60, TargetEnergyKWh: 12}
	vehicles := []model.Vehicle{{ID: "v1", MaxPower: 5}, {ID: "v2", MaxPower: 5}}
	date := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	avail := map[string][]AvailabilityWindow{
		"v1": {{Start: date, End: date.Add(24 * time.Hour)}},
		"v2": {{Start: date.Add(12 * time.Hour), End: date.Add(24 * time.Hour)}},
	}
	s := Scheduler{Config: cfg, Vehicles: vehicles, Availability: avail}
	plan, err := s.GeneratePlan(date)
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	slotDur := time.Duration(cfg.SlotDurationMinutes) * time.Minute
	total := 0.0
	for _, e := range plan {
		total += e.PowerKW * slotDur.Hours()
	}
	if math.Abs(total-cfg.TargetEnergyKWh) > 1e-6 {
		t.Fatalf("expected %.1f got %.3f", cfg.TargetEnergyKWh, total)
	}
	for _, e := range plan {
		if e.VehicleID == "v2" && e.TimeSlot.Before(date.Add(12*time.Hour)) {
			t.Fatalf("v2 scheduled outside availability")
		}
	}
}

func TestGeneratePlanInfeasible(t *testing.T) {
	cfg := SchedulerConfig{SlotDurationMinutes: 60, TargetEnergyKWh: 1000}
	vehicles := []model.Vehicle{{ID: "v1", MaxPower: 5}}
	date := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	avail := map[string][]AvailabilityWindow{
		"v1": {{Start: date, End: date.Add(24 * time.Hour)}},
	}
	s := Scheduler{Config: cfg, Vehicles: vehicles, Availability: avail}
	if _, err := s.GeneratePlan(date); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadConfig(t *testing.T) {
	data := "slot_duration_minutes: 60\ntarget_energy_kwh: 10\n"
	f := bytes.NewBufferString(data)
	cfg, err := DecodeConfig(f, "yaml")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.SlotDurationMinutes != 60 || cfg.TargetEnergyKWh != 10 {
		t.Fatalf("bad cfg %#v", cfg)
	}
}

func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/cfg.json"
	if err := os.WriteFile(path, []byte(`{"slot_duration_minutes":15,"target_energy_kwh":5}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.SlotDurationMinutes != 15 || cfg.TargetEnergyKWh != 5 {
		t.Fatalf("bad cfg %#v", cfg)
	}
	_, err = LoadConfig(path + ".txt")
	if err == nil {
		t.Fatalf("expected error for wrong ext")
	}
}

func TestLoadConfigYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	data := "slot_duration_minutes: 45\ntarget_energy_kwh: 6"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.SlotDurationMinutes != 45 || cfg.TargetEnergyKWh != 6 {
		t.Fatalf("bad cfg %#v", cfg)
	}
}

func TestDecodeErrors(t *testing.T) {
	if _, err := DecodeConfig(bytes.NewBufferString("{}"), "toml"); err == nil {
		t.Fatalf("expected error")
	}
	path := filepath.Join(t.TempDir(), "cfg.txt")
	if err := os.WriteFile(path, []byte("bad"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := DecodeConfig(bytes.NewBufferString(":"), "yaml"); err == nil {
		t.Fatalf("expected yaml error")
	}
}

func TestDecodeJSON(t *testing.T) {
	cfg, err := DecodeConfig(bytes.NewBufferString(`{"slot_duration_minutes":30,"target_energy_kwh":8}`), "json")
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	if cfg.SlotDurationMinutes != 30 || cfg.TargetEnergyKWh != 8 {
		t.Fatalf("bad cfg %#v", cfg)
	}
}

func TestGeneratePlanInsufficient(t *testing.T) {
	date := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	cfg := SchedulerConfig{SlotDurationMinutes: 60, TargetEnergyKWh: 50}
	vehicles := []model.Vehicle{{ID: "v1", MaxPower: 1}}
	avail := map[string][]AvailabilityWindow{"v1": {{Start: date, End: date.Add(24 * time.Hour)}}}
	s := Scheduler{Config: cfg, Vehicles: vehicles, Availability: avail}
	if _, err := s.GeneratePlan(date); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGeneratePlanNoVehicles(t *testing.T) {
	date := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	cfg := SchedulerConfig{SlotDurationMinutes: 60, TargetEnergyKWh: 1}
	s := Scheduler{Config: cfg}
	if _, err := s.GeneratePlan(date); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGeneratePlanShift(t *testing.T) {
	date := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	cfg := SchedulerConfig{SlotDurationMinutes: 60, TargetEnergyKWh: 2}
	vehicles := []model.Vehicle{{ID: "v1", MaxPower: 10}}
	avail := map[string][]AvailabilityWindow{
		"v1": {{Start: date.Add(12 * time.Hour), End: date.Add(24 * time.Hour)}},
	}
	s := Scheduler{Config: cfg, Vehicles: vehicles, Availability: avail}
	plan, err := s.GeneratePlan(date)
	if err != nil {
		t.Fatalf("shift: %v", err)
	}
	slotDur := time.Duration(cfg.SlotDurationMinutes) * time.Minute
	total := 0.0
	for _, e := range plan {
		if e.TimeSlot.Before(date.Add(12 * time.Hour)) {
			t.Fatalf("scheduled before availability")
		}
		total += e.PowerKW * slotDur.Hours()
	}
	if math.Abs(total-cfg.TargetEnergyKWh) > 1e-6 {
		t.Fatalf("energy mismatch %.3f", total)
	}
}

func TestGeneratePlanRedistribute(t *testing.T) {
	date := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	cfg := SchedulerConfig{SlotDurationMinutes: 60, TargetEnergyKWh: 12}
	vehicles := []model.Vehicle{{ID: "v1", MaxPower: 0.2}, {ID: "v2", MaxPower: 1}}
	avail := map[string][]AvailabilityWindow{
		"v1": {{Start: date, End: date.Add(24 * time.Hour)}},
		"v2": {{Start: date, End: date.Add(24 * time.Hour)}},
	}
	s := Scheduler{Config: cfg, Vehicles: vehicles, Availability: avail}
	plan, err := s.GeneratePlan(date)
	if err != nil {
		t.Fatalf("redistribute: %v", err)
	}
	// look at first slot allocations
	firstSlot := date
	var p1, p2 float64
	for _, e := range plan {
		if !e.TimeSlot.Equal(firstSlot) {
			continue
		}
		switch e.VehicleID {
		case "v1":
			p1 = e.PowerKW
		case "v2":
			p2 = e.PowerKW
		}
	}
	if math.Abs(p1-0.2) > 1e-6 {
		t.Fatalf("v1 cap not used %.3f", p1)
	}
	if math.Abs(p1+p2-0.5) > 1e-6 {
		t.Fatalf("total not redistributed %.3f", p1+p2)
	}
}
