package logging

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"
)

// SQLiteStore persists logs to a SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates the database at path and ensures schema.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	schema := `CREATE TABLE IF NOT EXISTS dispatch_logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        ts INTEGER,
        signal_type TEXT,
        record TEXT
    );`
	if _, err := db.Exec(schema); err != nil {
		if cerr := db.Close(); cerr != nil {
			return nil, fmt.Errorf("close db: %v (schema err: %w)", cerr, err)
		}
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

// Append writes the record to the database.
func (s *SQLiteStore) Append(ctx context.Context, rec LogRecord) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO dispatch_logs (ts, signal_type, record) VALUES (?, ?, ?)`,
		rec.Timestamp.Unix(), rec.Signal.Type.String(), string(b))
	return err
}

// Query returns records matching q.
func (s *SQLiteStore) Query(ctx context.Context, q LogQuery) ([]LogRecord, error) {
	var args []any
	query := `SELECT record FROM dispatch_logs WHERE 1=1`
	if !q.Start.IsZero() {
		query += ` AND ts >= ?`
		args = append(args, q.Start.Unix())
	}
	if !q.End.IsZero() {
		query += ` AND ts <= ?`
		args = append(args, q.End.Unix())
	}
	if q.SignalType != 0 {
		query += ` AND signal_type = ?`
		args = append(args, q.SignalType.String())
	}
	query += ` ORDER BY ts`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var res []LogRecord
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var r LogRecord
		if err := json.Unmarshal([]byte(data), &r); err != nil {
			return nil, fmt.Errorf("unmarshal record: %w", err)
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
				}
			}
			if !matched {
				if _, ok := r.Response.FallbackAssignments[q.VehicleID]; ok {
					matched = true
				}
			}
			if !matched {
				continue
			}
		}
		res = append(res, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

// Close closes the underlying database.
func (s *SQLiteStore) Close() error { return s.db.Close() }
