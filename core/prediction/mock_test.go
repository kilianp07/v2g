package prediction

import (
	"testing"
	"time"
)

func TestMockPredictionEngine_PredictAvailability(t *testing.T) {
	eng := MockPredictionEngine{Availability: map[string]float64{"v1": 0.7}}
	if eng.PredictAvailability("v1", time.Now()) != 0.7 {
		t.Fatalf("expected configured value")
	}
	if eng.PredictAvailability("v2", time.Now()) != 1 {
		t.Fatalf("expected default value 1")
	}
}

func TestMockPredictionEngine_ForecastSoC(t *testing.T) {
	eng := MockPredictionEngine{SoCForecasts: map[string][]float64{"v1": {0.5, 0.6}}}
	res := eng.ForecastSoC("v1", 0)
	if len(res) != 2 || res[0] != 0.5 || res[1] != 0.6 {
		t.Fatalf("unexpected forecast %v", res)
	}
	if eng.ForecastSoC("v2", 0) != nil {
		t.Fatalf("expected nil for unknown vehicle")
	}
}
