package prediction

import "time"

// PredictionEngine defines methods to forecast vehicle availability and SoC.
type PredictionEngine interface {
	// PredictAvailability returns the probability [0,1] that the vehicle will
	// be available at time t.
	PredictAvailability(vehicleID string, t time.Time) float64

	// ForecastSoC returns SoC forecasts for future time steps up to the given
	// horizon. The slice may be empty if no forecast is available.
	ForecastSoC(vehicleID string, horizon time.Duration) []float64
}
