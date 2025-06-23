package metrics

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartPromServer starts an HTTP server exposing Prometheus metrics on the given address.
// The server runs until the provided context is canceled.
// A dedicated ServeMux is used to avoid interfering with other handlers.
func StartPromServer(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("prom server shutdown: %v", err)
		}
		cancel()
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
