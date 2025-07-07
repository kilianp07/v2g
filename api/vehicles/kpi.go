package vehicles

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	eco "github.com/kilianp07/v2g/core/metrics/eco"
)

// NewKPIHandler exposes ecological KPIs via GET /api/vehicles/{id}/kpis.
func NewKPIHandler(store eco.Store, factor float64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/vehicles/")
		parts := strings.Split(path, "/")
		if len(parts) < 2 || parts[1] != "kpis" {
			http.NotFound(w, r)
			return
		}
		id := parts[0]
		start, _ := time.Parse(time.RFC3339, r.URL.Query().Get("start"))
		end, _ := time.Parse(time.RFC3339, r.URL.Query().Get("end"))
		if end.IsZero() {
			end = time.Now()
		}
		recs, err := store.Query(id, start, end)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		type out struct {
			Date        string  `json:"date"`
			InjectedKWh float64 `json:"injected_kwh"`
			CO2Avoided  float64 `json:"co2_avoided"`
			EnergyRatio float64 `json:"energy_ratio"`
		}
		outSlice := make([]out, len(recs))
		for i, r := range recs {
			outSlice[i] = out{
				Date:        r.Date.Format("2006-01-02"),
				InjectedKWh: r.InjectedKWh,
				CO2Avoided:  r.CO2Avoided(factor),
				EnergyRatio: r.EnergyRatio(),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(outSlice)
	})
}
