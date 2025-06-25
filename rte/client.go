package rte

import (
	"context"
	"net/http"
	"time"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/logger"
)

// RTEClient polls the official RTE API for flexibility signals.
type RTEClient struct {
	mgr      Manager
	log      logger.Logger
	client   *http.Client
	apiURL   string
	ticker   *time.Ticker
	interval time.Duration
}

// NewRTEClient creates a new RTE API client.
func NewRTEClient(cfg config.RTEClientConfig, m Manager) *RTEClient {
	if cfg.PollIntervalSeconds <= 0 {
		cfg.PollIntervalSeconds = 60
	}
	return &RTEClient{
		mgr:      m,
		log:      logger.New("rte-client"),
		client:   &http.Client{Timeout: 10 * time.Second},
		apiURL:   cfg.APIURL,
		interval: time.Duration(cfg.PollIntervalSeconds) * time.Second,
	}
}

// Start begins the polling loop.
func (c *RTEClient) Start(ctx context.Context) error {
	c.ticker = time.NewTicker(c.interval)
	defer c.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.ticker.C:
			if err := c.poll(ctx); err != nil {
				c.log.Errorf("poll error: %v", err)
			}
		}
	}
}

func (c *RTEClient) poll(ctx context.Context) error {
	// TODO: implement OAuth2 token retrieval and API polling
	c.log.Infof("polling RTE API at %s", c.apiURL)
	_ = ctx
	// future: fetch data, parse []Signal and dispatch each
	return nil
}
