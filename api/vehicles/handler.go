package vehicles

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kilianp07/v2g/core/prediction"
	vehiclestatus "github.com/kilianp07/v2g/core/vehiclestatus"
)

// NewStatusHandler returns an HTTP handler exposing vehicle status data via GET /api/vehicles/status.
func NewStatusHandler(store vehiclestatus.Store, pred prediction.PredictionEngine) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		f := vehiclestatus.Filter{
			FleetID: r.URL.Query().Get("fleet_id"),
			Site:    r.URL.Query().Get("site"),
			Cluster: r.URL.Query().Get("cluster"),
		}
		entries := store.List(f)
		for i := range entries {
			id := entries[i].VehicleID
			if pred != nil {
				fc := pred.ForecastSoC(id, time.Hour)
				if len(fc) > 0 {
					entries[i].ForecastedSoC = map[string]float64{}
					step := 15 * time.Minute
					for j, v := range fc {
						entries[i].ForecastedSoC[fmt.Sprintf("t+%dm", int(step.Minutes())*j)] = v
					}
					now := time.Now().UTC()
					entries[i].ForecastedPluginWindow = vehiclestatus.TimeWindow{Start: now, End: now.Add(time.Hour)}
					entries[i].NextDispatchWindow = vehiclestatus.TimeWindow{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour)}
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(entries); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
