package factory

import (
	"fmt"

	"github.com/kilianp07/v2g/connectors"
	wholesalemarket "github.com/kilianp07/v2g/connectors/clients/wholesaleMarket"
)

const (
	IDWholesaleMarket = "wholesale_market"
)

var (
	errUnknownClient = "unknown connector id: %s"
)

func NewRTEClient(id string) (connectors.RTEClient, error) {
	switch id {
	case IDWholesaleMarket:
		return &wholesalemarket.Client{}, nil
	default:
		return nil, fmt.Errorf(errUnknownClient, id)
	}

}
