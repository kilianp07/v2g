package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTokenAndSetAuthHeader(t *testing.T) {
	// Simple OAuth2 token endpoint returning a static token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"token123","token_type":"bearer","expires_in":3600}`))
	}))
	defer server.Close()

	cfg := Conf{ClientID: "id", ClientSecret: "secret", AuthURL: server.URL}
	client := NewClientCred(cfg)

	token, err := client.GetToken()
	if err != nil {
		t.Fatalf("GetToken returned error: %v", err)
	}
	if token != "token123" {
		t.Fatalf("unexpected token %s", token)
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	if err := client.SetAuthHeader(req); err != nil {
		t.Fatalf("SetAuthHeader returned error: %v", err)
	}
	if auth := req.Header.Get("Authorization"); auth == "" {
		t.Fatalf("Authorization header not set")
	}
}
