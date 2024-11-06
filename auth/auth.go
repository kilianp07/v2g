package auth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var authenticator *ClientCred

type ClientCred struct {
	conf  clientcredentials.Config
	token *oauth2.Token
}

func NewClientCred(conf Conf) {
	authenticator = &ClientCred{
		conf: conf.toOauth2Config(),
	}
}

func Authenticator() *ClientCred {
	return authenticator
}

// GetToken retrieves a valid access token. If the current token is valid, it returns the existing token.
// Otherwise, it requests a new token using the client credentials configuration.
// Returns the access token as a string and an error if the token retrieval fails.
func (c *ClientCred) GetToken() (string, error) {
	if c.token.Valid() {
		return c.token.AccessToken, nil
	}
	var err error
	c.token, err = c.conf.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	return c.token.AccessToken, nil
}

// ForceRefresh retrieves a new token using the client credentials configuration
// and updates the client's token. It returns the new access token as a string
// and an error if the token retrieval fails.
//
// Returns:
//   - string: The new access token.
//   - error: An error if the token retrieval fails.
func (c *ClientCred) ForceRefresh() (string, error) {
	var err error
	c.token, err = c.conf.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	return c.token.AccessToken, nil
}
