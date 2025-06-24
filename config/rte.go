package config

import "fmt"

// RTEClientConfig holds settings for the RTE API client.
type RTEClientConfig struct {
	APIURL              string `json:"api_url"`
	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret"`
	TokenURL            string `json:"token_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
}

// RTEMockConfig holds settings for the mock HTTP server.
type RTEMockConfig struct {
	Address string `json:"address"`
}

// RTEConfig defines configuration for the RTE connector.
type RTEConfig struct {
	Mode   string          `json:"mode"`
	Client RTEClientConfig `json:"client"`
	Mock   RTEMockConfig   `json:"mock"`
}

// SetDefaults sets default values for optional fields.
func (c *RTEConfig) SetDefaults() {
	if c.Mode == "client" && c.Client.PollIntervalSeconds <= 0 {
		c.Client.PollIntervalSeconds = 60
	}
}

// Validate checks that required fields are present depending on Mode.
func (c *RTEConfig) Validate() error {
	switch c.Mode {
	case "mock":
		if c.Mock.Address == "" {
			return fmt.Errorf("mock.address is required")
		}
	case "client":
		if c.Client.APIURL == "" {
			return fmt.Errorf("client.api_url is required")
		}
		if c.Client.ClientID == "" || c.Client.ClientSecret == "" {
			return fmt.Errorf("client credentials are required")
		}
		if c.Client.TokenURL == "" {
			return fmt.Errorf("client.token_url is required")
		}
		if c.Client.PollIntervalSeconds <= 0 {
			return fmt.Errorf("client.poll_interval_seconds must be greater than 0")
		}
	default:
		return fmt.Errorf("unknown mode %s", c.Mode)
	}
	return nil
}
