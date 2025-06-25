package rte

import (
	"context"
	"strings"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/dispatch"
	"github.com/kilianp07/v2g/model"
)

// Manager is the subset of dispatch.DispatchManager used by connectors.
type Manager interface {
	Dispatch(model.FlexibilitySignal, []model.Vehicle) dispatch.DispatchResult
}

// RTEConnector defines the behavior of a connector receiving RTE signals.
type RTEConnector interface {
	Start(ctx context.Context) error
}

// NewConnector creates a connector depending on cfg.Mode ("client" or "mock").
func NewConnector(cfg config.RTEConfig, m Manager) RTEConnector {
	switch strings.ToLower(cfg.Mode) {
	case "mock":
		return NewRTEServerMock(cfg.Mock, m)
	default:
		return NewRTEClient(cfg.Client, m)
	}
}
