package dispatch

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kilianp07/v2g/core/dispatch/logging"
	"github.com/kilianp07/v2g/core/model"
)

// NewLogHandler returns an HTTP handler exposing dispatch logs via GET /api/dispatch/logs.
// Requests must include an Authorization header with "Bearer <token>" when token is non-empty.
func NewLogHandler(store logging.LogStore, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		q := logging.LogQuery{}
		if s := r.URL.Query().Get("start"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				q.Start = t
			}
		}
		if s := r.URL.Query().Get("end"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				q.End = t
			}
		}
		q.VehicleID = r.URL.Query().Get("vehicle_id")
		if st := r.URL.Query().Get("signal_type"); st != "" {
			if v, ok := signalTypeFromString(st); ok {
				q.SignalType = v
			}
		}
		records, err := store.Query(r.Context(), q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(records); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func signalTypeFromString(s string) (model.SignalType, bool) {
	switch s {
	case "FCR":
		return model.SignalFCR, true
	case "aFRR":
		return model.SignalAFRR, true
	case "MA":
		return model.SignalMA, true
	case "NEBEF":
		return model.SignalNEBEF, true
	case "EcoWatt":
		return model.SignalEcoWatt, true
	default:
		return 0, false
	}
}
