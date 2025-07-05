package mqtt

import "errors"

// ErrAckTimeout is returned when no acknowledgment is received before the timeout.
var ErrAckTimeout = errors.New("timeout waiting for ack")
