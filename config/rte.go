package config

// RTEConfig defines configuration for the RTE connector.
type RTEConfig struct {
	Mode                string `json:"mode"`
	MockAddress         string `json:"mock_address"`
	APIURL              string `json:"api_url"`
	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret"`
	TokenURL            string `json:"token_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
}
