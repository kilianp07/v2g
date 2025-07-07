package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// RotatingJSONLStore stores logs in a JSONL file with automatic rotation.
type RotatingJSONLStore struct {
	logger *lumberjack.Logger
	path   string
}

// NewRotatingJSONLStore creates a store with rotation options in megabytes and days.
func NewRotatingJSONLStore(path string, maxSizeMB, maxBackups, maxAgeDays int) (*RotatingJSONLStore, error) {
	lj := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAgeDays,
		Compress:   false,
	}
	// ensure directory exists
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return &RotatingJSONLStore{logger: lj, path: path}, nil
}

// Append writes the record and triggers rotation if needed.
func (s *RotatingJSONLStore) Append(ctx context.Context, rec LogRecord) error {
	_ = ctx
	enc := json.NewEncoder(s.logger)
	return enc.Encode(rec)
}

// Query reads all log files including rotated ones.
//
//gocyclo:ignore
func (s *RotatingJSONLStore) Query(ctx context.Context, q LogQuery) ([]LogRecord, error) {
	_ = ctx
	pattern := s.path + "*"
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var res []LogRecord
	for _, f := range files {
		file, err := os.Open(f)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var r LogRecord
			if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
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
			if q.VehicleID != "" {
				matched := false
				for _, id := range r.VehiclesSelected {
					if id == q.VehicleID {
						matched = true
						break
					}
				}
				if !matched {
					if _, ok := r.Response.Assignments[q.VehicleID]; ok {
						matched = true
					} else if _, ok := r.Response.FallbackAssignments[q.VehicleID]; ok {
						matched = true
					}
				}
				if !matched {
					continue
				}
			}
			res = append(res, r)
		}
		_ = file.Close()
	}
	return res, nil
}

// Close closes the underlying writer.
func (s *RotatingJSONLStore) Close() error {
	return s.logger.Close()
}
