package kpi

import (
	"database/sql"
	"time"

	core "github.com/kilianp07/v2g/core/metrics/eco"
	_ "modernc.org/sqlite"
)

// SQLiteStore persists KPI records in a SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates the database and ensures schema.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	schema := `CREATE TABLE IF NOT EXISTS eco_kpi (
        vehicle_id TEXT,
        day INTEGER,
        injected REAL,
        consumed REAL,
        PRIMARY KEY(vehicle_id, day)
    );`
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

// Add inserts or updates the KPI record.
func (s *SQLiteStore) Add(r core.Record) error {
	d := core.Day(r.Date)
	_, err := s.db.Exec(`INSERT INTO eco_kpi (vehicle_id, day, injected, consumed)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(vehicle_id, day) DO UPDATE SET
            injected = injected + excluded.injected,
            consumed = consumed + excluded.consumed`,
		r.VehicleID, d.Unix(), r.InjectedKWh, r.ConsumedKWh)
	return err
}

// Query returns records in the range [start,end].
func (s *SQLiteStore) Query(vehicleID string, start, end time.Time) ([]core.Record, error) {
	start = core.Day(start)
	end = core.Day(end)
	rows, err := s.db.Query(`SELECT vehicle_id, day, injected, consumed
        FROM eco_kpi WHERE vehicle_id = ? AND day >= ? AND day <= ? ORDER BY day`,
		vehicleID, start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var res []core.Record
	for rows.Next() {
		var vid string
		var ts int64
		var inj, cons float64
		if err := rows.Scan(&vid, &ts, &inj, &cons); err != nil {
			return nil, err
		}
		res = append(res, core.Record{
			VehicleID:   vid,
			Date:        time.Unix(ts, 0).UTC(),
			InjectedKWh: inj,
			ConsumedKWh: cons,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

// Close closes the underlying database.
func (s *SQLiteStore) Close() error { return s.db.Close() }
