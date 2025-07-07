package eco

import "time"

// Store persists ecological KPI records.
type Store interface {
	Add(Record) error
	Query(vehicleID string, start, end time.Time) ([]Record, error)
}

// Helper to align time to start of day in UTC.
func Day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
