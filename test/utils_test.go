package test

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kilianp07/v2g/rte"
)

// waitForRTEServer checks readiness of an RTEServerMock by polling its ping endpoint.
func waitForRTEServer(s *rte.RTEServerMock, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		url := "http://" + s.Addr() + "/rte/ping"
		resp, err := http.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			if err := resp.Body.Close(); err != nil {
				return err
			}
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("server not ready: %s", s.Addr())
}
