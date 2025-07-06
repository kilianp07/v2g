package logging

import (
	"context"
	"time"

	"github.com/kilianp07/v2g/core/model"
)

// LogRecord captures one dispatch decision and result.
type LogRecord struct {
	Timestamp        time.Time               `json:"timestamp"`
	Signal           model.FlexibilitySignal `json:"signal"`
	TargetPower      float64                 `json:"target_power"`
	VehiclesSelected []string                `json:"vehicles_selected"`
	Response         Result                  `json:"response"`
}

// Result mirrors dispatch.DispatchResult for logging purposes.
type Result struct {
	Assignments         map[string]float64      `json:"assignments"`
	FallbackAssignments map[string]float64      `json:"fallback_assignments"`
	Errors              map[string]string       `json:"errors"`
	Acknowledged        map[string]bool         `json:"acknowledged"`
	Signal              model.FlexibilitySignal `json:"signal"`
	MarketPrice         float64                 `json:"market_price"`
	Scores              map[string]float64      `json:"scores"`
}

// LogQuery defines filters for retrieving records.
type LogQuery struct {
	Start      time.Time
	End        time.Time
	VehicleID  string
	SignalType model.SignalType
}

// LogStore persists LogRecords and supports querying.
type LogStore interface {
	Append(ctx context.Context, rec LogRecord) error
	Query(ctx context.Context, q LogQuery) ([]LogRecord, error)
	Close() error
}
