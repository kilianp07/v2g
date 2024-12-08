package wholesalemarket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kilianp07/v2g/auth"
	"github.com/kilianp07/v2g/connectors"
)

var (
	wholesaleBaseURL = "https://digital.iservices.rte-france.com/open_api/wholesale_market/v2/france_power_exchanges?start_date=%s&end_date=%s"
)

type Client struct {
	startDate time.Time
	endDate   time.Time
}

// Fetch retrieves the wholesale market data for the specified date range.
// It requires an authentication client and exactly two options to be set.
//
// Parameters:
//   - authClient: An instance of auth.ClientCred used to set the authorization header.
//   - opts: A variadic list of connectors.Option, exactly two options must be provided.
//
// Returns:
//   - connectors.RTEResponse: The response from the wholesale market API.
//   - error: An error if the request fails or the response cannot be processed.
//
// Errors:
//   - Returns an error if the number of options provided is not exactly two.
//   - Returns an error if the request creation fails.
//   - Returns an error if setting the authorization header fails.
//   - Returns an error if the request fails to execute.
//   - Returns an error if the response status code is not 200 OK.
//   - Returns an error if the response body cannot be read or decoded.
func (w *Client) Fetch(authClient *auth.ClientCred, opts ...connectors.Option) (connectors.RTEResponse, error) {
	client := &http.Client{}

	if len(opts) != 2 {
		return nil, fmt.Errorf("missing options: %d are set", len(opts))
	}

	for _, opt := range opts {
		if err := opt(w); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf(wholesaleBaseURL, w.startDate.Format(time.RFC3339), w.endDate.Format(time.RFC3339))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	err = authClient.SetAuthHeader(req)
	if err != nil {
		return nil, fmt.Errorf("failed to set auth header: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	var marketResponse Response
	if err := json.Unmarshal(body, &marketResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &marketResponse, nil
}
