package eco

import (
	"sort"
	"sync"
	"time"
)

// MemoryStore stores records in memory for testing or lightweight usage.
type MemoryStore struct {
	mu   sync.Mutex
	data map[string]map[time.Time]*Record
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: map[string]map[time.Time]*Record{}}
}

// Add inserts or updates the record aggregated by day and vehicle.
func (s *MemoryStore) Add(r Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[r.VehicleID] == nil {
		s.data[r.VehicleID] = map[time.Time]*Record{}
	}
	d := Day(r.Date)
	rec := s.data[r.VehicleID][d]
	if rec == nil {
		rec = &Record{VehicleID: r.VehicleID, Date: d}
		s.data[r.VehicleID][d] = rec
	}
	rec.InjectedKWh += r.InjectedKWh
	rec.ConsumedKWh += r.ConsumedKWh
	return nil
}

// Query returns records between start and end inclusive.
func (s *MemoryStore) Query(vehicleID string, start, end time.Time) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	start = Day(start)
	end = Day(end)
	var res []Record
	m := s.data[vehicleID]
	for d, r := range m {
		if d.Before(start) || d.After(end) {
			continue
		}
		res = append(res, *r)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Date.Before(res[j].Date) })
	return res, nil
}
