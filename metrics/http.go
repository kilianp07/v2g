package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartPromServer starts an HTTP server exposing Prometheus metrics on the given address.
// A dedicated ServeMux is used to avoid interfering with other handlers.
func StartPromServer(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, mux)
}
