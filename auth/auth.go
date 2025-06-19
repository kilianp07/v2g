package auth

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type ClientCred struct {
	conf  clientcredentials.Config
	token *oauth2.Token
}

func NewClientCred(conf Conf) *ClientCred {
	return &ClientCred{
		conf: conf.toOauth2Config(),
	}
}

// GetToken retrieves a valid access token. If the current token is valid, it returns the existing token.
// Otherwise, it requests a new token using the client credentials configuration.
// Returns the access token as a string and an error if the token retrieval fails.
func (c *ClientCred) GetToken() (string, error) {
	if c.token != nil && c.token.Valid() {
		return c.token.AccessToken, nil
	}
	if err := c.getToken(); err != nil {
		return "", err
	}
	return c.token.AccessToken, nil
}

func (c *ClientCred) getToken() error {
	var err error
	c.token, err = c.conf.Token(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	return nil
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
func (c *ClientCred) SetAuthHeader(r *http.Request) error {
	if c.token != nil && c.token.Valid() {
		c.token.SetAuthHeader(r)
		return nil
	}

	if err := c.getToken(); err != nil {
		return err
	}
	c.token.SetAuthHeader(r)
	return nil
}
