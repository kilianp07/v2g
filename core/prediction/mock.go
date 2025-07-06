package prediction

import "time"

// MockPredictionEngine returns deterministic availability and SoC forecasts.
type MockPredictionEngine struct {
	Availability map[string]float64
	SoCForecasts map[string][]float64
}

// PredictAvailability returns the configured probability for the vehicle or 1.0.
func (m MockPredictionEngine) PredictAvailability(id string, t time.Time) float64 {
	_ = t
	if m.Availability == nil {
		return 1
	}
	if v, ok := m.Availability[id]; ok {
		return v
	}
	return 1
}

// ForecastSoC returns the configured forecast slice for the vehicle or an empty slice.
func (m MockPredictionEngine) ForecastSoC(id string, h time.Duration) []float64 {
	_ = h
	if m.SoCForecasts == nil {
		return nil
	}
	if s, ok := m.SoCForecasts[id]; ok {
		cp := make([]float64, len(s))
		copy(cp, s)
		return cp
	}
	return nil
}
