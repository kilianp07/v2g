package scenarios

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kilianp07/v2g/core/model"
)

func TestScenario(t *testing.T) {
	files, err := filepath.Glob("*.yaml")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		sc, err := Load(f)
		if err != nil {
			t.Fatalf("load %s: %v", f, err)
		}
		t.Run(sc.Name, func(t *testing.T) {
			RunScenario(t, sc)
		})
	}
}

func TestParseSignalType(t *testing.T) {
	cases := map[string]int{
		"FCR":     int(parseSignalType("FCR")),
		"aFRR":    int(parseSignalType("aFRR")),
		"MA":      int(parseSignalType("MA")),
		"NEBEF":   int(parseSignalType("NEBEF")),
		"EcoWatt": int(parseSignalType("EcoWatt")),
	}
	for s, v := range cases {
		if v < 0 {
			t.Errorf("%s parsed negative", s)
		}
	}
}

func TestLoadInvalid(t *testing.T) {
	if _, err := Load("no-file.yaml"); err == nil {
		t.Fatal("expected error for missing file")
	}
	tmp, err := os.CreateTemp(t.TempDir(), "bad*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(":"); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(tmp.Name()); err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestRemoveVehicle(t *testing.T) {
	vehicles := []model.Vehicle{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	out := removeVehicle(vehicles, "b")
	if len(out) != 2 {
		t.Fatalf("expected 2 vehicles, got %d", len(out))
	}
	if out[0].ID != "a" || out[1].ID != "c" {
		t.Fatalf("unexpected order")
	}
}
