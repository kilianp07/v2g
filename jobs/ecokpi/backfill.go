package ecokpi

import (
	dispatch "github.com/kilianp07/v2g/core/dispatch"
	eco "github.com/kilianp07/v2g/core/metrics/eco"
)

// Backfill processes historical dispatch results and populates the store.
func Backfill(store eco.Store, history []dispatch.DispatchResult) error {
	for _, h := range history {
		for vid, p := range h.Assignments {
			kwh := p * h.Signal.Duration.Hours()
			rec := eco.Record{VehicleID: vid, Date: eco.Day(h.Signal.Timestamp)}
			if p >= 0 {
				rec.InjectedKWh = kwh
			} else {
				rec.ConsumedKWh = -kwh
			}
			if err := store.Add(rec); err != nil {
				return err
			}
		}
	}
	return nil
}
