package vehiclestatus

import (
	"sort"
	"sync"
	"time"
)

type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// LastDispatch mirrors the summary of a dispatch decision.
type LastDispatch struct {
	SignalType       string    `json:"signal_type"`
	TargetPower      float64   `json:"target_power"`
	VehiclesSelected []string  `json:"vehicles_selected"`
	Timestamp        time.Time `json:"timestamp"`
}

// Status captures the current known state of a vehicle.
type Status struct {
	VehicleID              string             `json:"vehicle_id"`
	FleetID                string             `json:"fleet_id,omitempty"`
	Site                   string             `json:"site,omitempty"`
	Cluster                string             `json:"cluster,omitempty"`
	CurrentStatus          string             `json:"current_status"`
	ForecastedPluginWindow TimeWindow         `json:"forecasted_plugin_window,omitempty"`
	ForecastedSoC          map[string]float64 `json:"forecasted_soc,omitempty"`
	NextDispatchWindow     TimeWindow         `json:"next_dispatch_window,omitempty"`
	LastDispatchDecision   LastDispatch       `json:"last_dispatch_decision"`
}

type Filter struct {
	FleetID string
	Site    string
	Cluster string
}

type Store interface {
	Set(Status)
	List(Filter) []Status
	RecordDispatch(id string, dec LastDispatch)
}

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]Status
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: map[string]Status{}}
}

func (s *MemoryStore) Set(st Status) {
	s.mu.Lock()
	s.data[st.VehicleID] = st
	s.mu.Unlock()
}

func (s *MemoryStore) RecordDispatch(id string, dec LastDispatch) {
	s.mu.Lock()
	st := s.data[id]
	if st.VehicleID == "" {
		st.VehicleID = id
	}
	st.LastDispatchDecision = dec
	st.CurrentStatus = "dispatched"
	s.data[id] = st
	s.mu.Unlock()
}

func (s *MemoryStore) List(f Filter) []Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]Status, 0, len(s.data))
	for _, st := range s.data {
		if f.FleetID != "" && st.FleetID != f.FleetID {
			continue
		}
		if f.Site != "" && st.Site != f.Site {
			continue
		}
		if f.Cluster != "" && st.Cluster != f.Cluster {
			continue
		}
		res = append(res, st)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].VehicleID < res[j].VehicleID })
	return res
}
