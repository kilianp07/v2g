package export

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/kilianp07/v2g/core/scheduler"
)

// WriteJSON writes the effacement plan to w in JSON format.
func WriteJSON(w io.Writer, entries []scheduler.EffacementEntry) error {
	enc := json.NewEncoder(w)
	return enc.Encode(entries)
}

// WriteCSV writes the effacement plan to w in CSV format with RTE headers.
func WriteCSV(w io.Writer, entries []scheduler.EffacementEntry) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"vehicle_id", "timeslot", "power_kw"}); err != nil {
		return err
	}
	for _, e := range entries {
		rec := []string{
			e.VehicleID,
			e.TimeSlot.Format(time.RFC3339),
			strconv.FormatFloat(e.PowerKW, 'f', -1, 64),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
