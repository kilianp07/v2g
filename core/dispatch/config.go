package dispatch

// Config defines dispatch-related settings.
type Config struct {
	AckTimeoutSeconds int `json:"ack_timeout_seconds"`
}
