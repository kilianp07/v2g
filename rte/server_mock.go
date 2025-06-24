package rte

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/logger"
	"github.com/kilianp07/v2g/model"
)

// RTEServerMock exposes HTTP endpoints for injecting signals locally.
type RTEServerMock struct {
	addr   string
	mgr    Manager
	log    logger.Logger
	srv    *http.Server
	total  *prometheus.CounterVec
	failed prometheus.Counter
}

// NewRTEServerMock creates a new mock server using the default Prometheus
// registerer.
func NewRTEServerMock(cfg config.RTEMockConfig, m Manager, log logger.Logger) *RTEServerMock {
	return NewRTEServerMockWithRegistry(cfg, m, log, prometheus.DefaultRegisterer)
}

// NewRTEServerMockWithRegistry creates a new mock server and registers metrics on
// the provided registerer. If reg is nil the default registerer is used.
func NewRTEServerMockWithRegistry(cfg config.RTEMockConfig, m Manager, log logger.Logger, reg prometheus.Registerer) *RTEServerMock {
	if log == nil {
		log = logger.NopLogger{}
	}
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	total := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "rte_signals_total",
		Help: "Total received RTE signals",
	}, []string{"signal_type"})
	failed := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rte_signals_failed",
		Help: "Failed RTE signals",
	})

	if err := reg.Register(total); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			total = are.ExistingCollector.(*prometheus.CounterVec)
		}
	}
	if err := reg.Register(failed); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			failed = are.ExistingCollector.(prometheus.Counter)
		}
	}

	return &RTEServerMock{
		addr:   cfg.Address,
		mgr:    m,
		log:    log,
		total:  total,
		failed: failed,
	}
}

func (s *RTEServerMock) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rte/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("pong")); err != nil {
			s.log.Errorf("write pong: %v", err)
		}
	})
	mux.HandleFunc("/rte/signal", s.handleSignal)
	return mux
}

func (s *RTEServerMock) handleSignal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var sig Signal
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		s.failed.Inc()
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := sig.Validate(); err != nil {
		s.failed.Inc()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fs, err := sig.ToFlexibility()
	if err != nil {
		s.failed.Inc()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.total.WithLabelValues(sig.SignalType).Inc()
	s.log.Infof("dispatching %s signal", sig.SignalType)
	s.mgr.Dispatch(fs, []model.Vehicle{})
	w.WriteHeader(http.StatusOK)
}

// Addr returns the listening address once Start has been called.
func (s *RTEServerMock) Addr() string { return s.addr }

// Start runs the HTTP server until the context is canceled.
func (s *RTEServerMock) Start(ctx context.Context) error {
	mux := s.routes()
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.addr = ln.Addr().String()
	s.srv = &http.Server{Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.log.Errorf("shutdown server: %v", err)
		}
		cancel()
	}()
	s.log.Infof("RTE mock server listening on %s", s.addr)
	err = s.srv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
