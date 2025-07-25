package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"sync"
)

func recordMatchesVehicle(r LogRecord, id string) bool {
	if id == "" {
		return true
	}
	for _, vid := range r.VehiclesSelected {
		if vid == id {
			return true
		}
	}
	if _, ok := r.Response.Assignments[id]; ok {
		return true
	}
	if _, ok := r.Response.FallbackAssignments[id]; ok {
		return true
	}
	return false
}

// JSONLStore stores logs in a JSONL file.
type JSONLStore struct {
	path string
	mu   sync.Mutex
}

func NewJSONLStore(path string) (*JSONLStore, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if cerr := f.Close(); cerr != nil {
		return nil, cerr
	}
	return &JSONLStore{path: path}, nil
}

func (s *JSONLStore) Append(ctx context.Context, rec LogRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	return enc.Encode(rec)
}

func (s *JSONLStore) Query(ctx context.Context, q LogQuery) ([]LogRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var res []LogRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var r LogRecord
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		if !q.Start.IsZero() && r.Timestamp.Before(q.Start) {
			continue
		}
		if !q.End.IsZero() && r.Timestamp.After(q.End) {
			continue
		}
		if q.SignalType != 0 && r.Signal.Type != q.SignalType {
			continue
		}
		if !recordMatchesVehicle(r, q.VehicleID) {
			continue
		}
		res = append(res, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *JSONLStore) Close() error { return nil }
