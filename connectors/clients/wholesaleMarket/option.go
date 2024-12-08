package wholesalemarket

import (
	"fmt"
	"time"

	"github.com/kilianp07/v2g/connectors"
)

func WithStartDate(startDate time.Time) connectors.Option {
	return func(c connectors.RTEClient) error {
		if w, ok := c.(*Client); ok {
			w.startDate = startDate
			return nil
		}
		return fmt.Errorf(connectors.ErrIncompatipleOption, "WithStartDate", "wholesale_market")
	}
}

func WithEndDate(endDate time.Time) connectors.Option {
	return func(c connectors.RTEClient) error {
		if w, ok := c.(*Client); ok {
			w.endDate = endDate
			return nil
		}
		return fmt.Errorf(connectors.ErrIncompatipleOption, "WithEndDate", "wholesale_market")
	}
}
