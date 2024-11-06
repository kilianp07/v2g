package auth

import "golang.org/x/oauth2/clientcredentials"

// Conf represents the configuration needed for authentication.
// It includes the client ID, client secret, and the authentication URL.
type Conf struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthURL      string `json:"auth_url"`
}

func (c *Conf) toOauth2Config() clientcredentials.Config {
	return clientcredentials.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		TokenURL:     c.AuthURL,
	}
}
