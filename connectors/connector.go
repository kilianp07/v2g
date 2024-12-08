package connectors

import (
	"github.com/kilianp07/v2g/auth"
)

type RTEClient interface {
	Fetch(authClient *auth.ClientCred, opts ...Option) (RTEResponse, error)
}

type RTEResponse interface {
	PriceChartHTML() (string, error)
}
