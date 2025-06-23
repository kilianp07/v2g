package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartPromServer starts an HTTP server exposing Prometheus metrics on the given address.
// It registers a /metrics endpoint using the default registry.
func StartPromServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}
