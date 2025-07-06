package main

import (
	"math/rand"
	"testing"
	"time"
)

func TestGenerateFleetCount(t *testing.T) {
	fleetRng = rand.New(rand.NewSource(1))
	cfg := FleetConfig{Size: 5}
	vs := GenerateFleet(cfg, nil)
	if len(vs) != 5 {
		t.Fatalf("expected 5 vehicles, got %d", len(vs))
	}
	if vs[0].ID != "veh0001" || vs[4].ID != "veh0005" {
		t.Fatalf("unexpected ids %s %s", vs[0].ID, vs[4].ID)
	}
}

func TestDistribution(t *testing.T) {
	fleetRng = rand.New(rand.NewSource(1))
	cfg := FleetConfig{Size: 100, CommuterPct: 0.6}
	vs := GenerateFleet(cfg, nil)
	commuters := 0
	for i := range vs {
		if vs[i].Segment == "commuter" {
			commuters++
		}
	}
	if commuters < 40 || commuters > 80 {
		t.Fatalf("commuter ratio unexpected: %d", commuters)
	}
}

func TestLoadAvailability(t *testing.T) {
	data := []byte(`{"0":0.1,"1":0.2,"2":0.3}`)
	prof, err := LoadAvailabilityProfile(data)
	if err != nil {
		t.Fatal(err)
	}
	if prof[2] != 0.3 {
		t.Fatalf("expected 0.3 got %f", prof[2])
	}
}

func TestLoadAvailabilityError(t *testing.T) {
	_, err := LoadAvailabilityProfile([]byte(`invalid`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScheduleOverride(t *testing.T) {
	cfg := FleetConfig{Size: 1, Schedule: map[string]time.Time{"veh0001": time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)}}
	vs := GenerateFleet(cfg, nil)
	if vs[0].Departure.IsZero() {
		t.Fatal("departure not set")
	}
}

func TestTemplateOverride(t *testing.T) {
	tmpl := map[string]VehicleTemplate{
		"veh0002": {Departure: "2024-01-01T09:00:00Z"},
	}
	cfg := FleetConfig{Size: 3}
	vs := GenerateFleet(cfg, tmpl)
	if vs[1].Departure.IsZero() {
		t.Fatal("template departure not applied")
	}
}
