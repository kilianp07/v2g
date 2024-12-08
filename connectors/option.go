package connectors

var (
	ErrIncompatipleOption = "option %s is not compatible with %s client"
)

type Option func(RTEClient) error
