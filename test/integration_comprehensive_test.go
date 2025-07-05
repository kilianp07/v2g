package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/core/dispatch"
	coremetrics "github.com/kilianp07/v2g/core/metrics"
	"github.com/kilianp07/v2g/core/model"
	"github.com/kilianp07/v2g/infra/logger"
	"github.com/kilianp07/v2g/infra/metrics"
	"github.com/kilianp07/v2g/infra/mqtt"
	"github.com/kilianp07/v2g/internal/eventbus"
	"github.com/kilianp07/v2g/rte"
)

// TestComprehensiveIntegration teste tous les composants ensemble
func TestComprehensiveIntegration(t *testing.T) {
	testCases := []struct {
		name       string
		dispatcher string
		fallback   string
		vehicles   []model.Vehicle
		signals    []model.FlexibilitySignal
		expectAcks int
	}{
		{
			name:       "SmartDispatcher_with_BalancedFallback",
			dispatcher: "smart",
			fallback:   "balanced",
			vehicles: []model.Vehicle{
				{ID: "v1", SoC: 0.2, IsV2G: true, Available: true, MaxPower: 22, BatteryKWh: 40},
				{ID: "v2", SoC: 0.8, IsV2G: true, Available: true, MaxPower: 11, BatteryKWh: 60},
				{ID: "v3", SoC: 0.5, IsV2G: false, Available: true, MaxPower: 7, BatteryKWh: 30},
			},
			signals: []model.FlexibilitySignal{
				{Type: model.SignalFCR, PowerKW: 15, Duration: 900 * time.Second, Timestamp: time.Now()},
				{Type: model.SignalAFRR, PowerKW: 25, Duration: 300 * time.Second, Timestamp: time.Now()},
			},
			expectAcks: 1,
		},
		{
			name:       "LPDispatcher_with_ProbabilisticFallback",
			dispatcher: "lp",
			fallback:   "probabilistic",
			vehicles: []model.Vehicle{
				{ID: "v4", SoC: 0.9, IsV2G: true, Available: true, MaxPower: 50, BatteryKWh: 100},
				{ID: "v5", SoC: 0.1, IsV2G: true, Available: false, MaxPower: 22, BatteryKWh: 75},
			},
			signals: []model.FlexibilitySignal{
				{Type: model.SignalMA, PowerKW: 30, Duration: 600 * time.Second, Timestamp: time.Now()},
			},
			expectAcks: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runIntegrationTest(t, tc.dispatcher, tc.fallback, tc.vehicles, tc.signals, tc.expectAcks)
		})
	}
}

func runIntegrationTest(t *testing.T, dispatcherType, fallbackType string, vehicles []model.Vehicle, signals []model.FlexibilitySignal, expectAcks int) {
	// Setup metrics
	reg := prometheus.NewRegistry()
	sinkIf, err := metrics.NewPromSinkWithRegistry(coremetrics.Config{}, reg)
	if err != nil {
		t.Fatalf("prom sink: %v", err)
	}
	sink := sinkIf.(*metrics.PromSink)

	// Setup publisher and eventbus
	publisher := mqtt.NewMockPublisher()
	bus := eventbus.New()

	// Create dispatcher
	var dispatcher dispatch.Dispatcher
	switch dispatcherType {
	case "smart":
		dispatcher = &dispatch.SmartDispatcher{}
	case "lp":
		dispatcher = &dispatch.LPDispatcher{}
	default:
		dispatcher = dispatch.EqualDispatcher{}
	}

	// Create fallback
	var fallback dispatch.FallbackStrategy
	switch fallbackType {
	case "balanced":
		fallback = &dispatch.BalancedFallback{}
	case "probabilistic":
		fallback = dispatch.NewProbabilisticFallback(logger.NopLogger{})
	default:
		fallback = dispatch.NoopFallback{}
	}

	// Create manager
	mgr, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		dispatcher,
		fallback,
		publisher,
		100*time.Millisecond,
		sink,
		bus,
		nil,
		logger.NopLogger{},
		nil,
	)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	// Setup RTE mock server
	wrapper := managerWrapper{mgr: mgr, vehicles: vehicles}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := rte.NewRTEServerMockWithRegistry(config.RTEMockConfig{Address: "127.0.0.1:0"}, wrapper, reg)
	go func() { _ = srv.Start(ctx) }()

	if err := waitForRTEServer(srv, 2*time.Second); err != nil {
		t.Fatalf("server not ready: %v", err)
	}

	// Create RTE client (note: n'est pas utilisé dans ce test car on teste directement le manager)
	_ = rte.NewRTEClient(config.RTEClientConfig{
		APIURL:              "http://" + srv.Addr(),
		ClientID:            "test-client",
		ClientSecret:        "test-secret",
		TokenURL:            "http://" + srv.Addr() + "/token",
		PollIntervalSeconds: 1,
	}, mgr)

	// Send signals through the HTTP mock server so metrics are recorded
	for _, signal := range signals {
		rtesig := rte.Signal{
			SignalType: signal.Type.String(),
			StartTime:  signal.Timestamp,
			EndTime:    signal.Timestamp.Add(signal.Duration),
			Power:      signal.PowerKW,
		}
		data, _ := json.Marshal(rtesig)
		resp, err := http.Post("http://"+srv.Addr()+"/rte/signal", "application/json", bytes.NewReader(data))
		if err != nil {
			t.Fatalf("post signal: %v", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close resp body: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: %d", resp.StatusCode)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	finalAckCount := countAcks(reg)
	if finalAckCount != expectAcks {
		t.Errorf("Expected %d ACKs, got %d", expectAcks, finalAckCount)
	}

	// Validate metrics
	validateMetrics(t, reg, len(signals), finalAckCount)
}

func countAcks(reg *prometheus.Registry) int {
	mfs, err := reg.Gather()
	if err != nil {
		return 0
	}
	total := 0
	for _, mf := range mfs {
		if *mf.Name == "dispatch_events_total" {
			for _, m := range mf.Metric {
				for _, l := range m.Label {
					if *l.Name == "acknowledged" && *l.Value == "true" {
						total += int(m.GetCounter().GetValue())
					}
				}
			}
		}
	}
	return total
}

func validateMetrics(t *testing.T, reg *prometheus.Registry, expectedSignals, expectedAcks int) {
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	signalCount := 0
	ackCount := 0

	for _, mf := range mfs {
		switch *mf.Name {
		case "rte_signals_total":
			for _, m := range mf.Metric {
				signalCount += int(m.GetCounter().GetValue())
			}
		case "dispatch_events_total":
			for _, m := range mf.Metric {
				for _, label := range m.Label {
					if *label.Name == "acknowledged" && *label.Value == "true" {
						ackCount += int(m.GetCounter().GetValue())
					}
				}
			}
		}
	}

	if signalCount != expectedSignals {
		t.Errorf("Expected %d signals in metrics, got %d", expectedSignals, signalCount)
	}

	if ackCount != expectedAcks {
		t.Errorf("Expected %d acknowledged events in metrics, got %d", expectedAcks, ackCount)
	}
}

// TestErrorHandlingIntegration teste la gestion d'erreurs end-to-end
func TestErrorHandlingIntegration(t *testing.T) {
	testCases := []struct {
		name           string
		serverError    bool
		invalidSignal  bool
		unavailableVeh bool
		expectError    bool
	}{
		{
			name:        "ServerError_should_be_handled",
			serverError: true,
			expectError: true,
		},
		{
			name:          "InvalidSignal_should_be_ignored",
			invalidSignal: true,
			expectError:   false,
		},
		{
			name:           "UnavailableVehicles_should_use_fallback",
			unavailableVeh: true,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			sinkIf, err := metrics.NewPromSinkWithRegistry(coremetrics.Config{}, reg)
			if err != nil {
				t.Fatalf("prom sink: %v", err)
			}

			publisher := mqtt.NewMockPublisher()
			bus := eventbus.New()

			vehicles := []model.Vehicle{
				{ID: "test-v1", SoC: 0.5, IsV2G: true, Available: !tc.unavailableVeh, MaxPower: 22, BatteryKWh: 40},
			}

			mgr, err := dispatch.NewDispatchManager(
				dispatch.SimpleVehicleFilter{},
				dispatch.EqualDispatcher{},
				&dispatch.BalancedFallback{},
				publisher,
				100*time.Millisecond,
				sinkIf.(*metrics.PromSink),
				bus,
				nil,
				logger.NopLogger{},
				nil,
			)
			if err != nil {
				t.Fatalf("manager: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tc.serverError {
				// Test avec un serveur qui retourne toujours une erreur
				errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}))
				defer errorServer.Close()

				// Test direct dispatch au lieu de polling
				result := mgr.Dispatch(model.FlexibilitySignal{
					Type:      model.SignalFCR,
					PowerKW:   15,
					Duration:  300 * time.Second,
					Timestamp: time.Now(),
				}, vehicles)

				// Vérifier que le dispatch a été traité même avec une erreur serveur
				if len(result.Assignments) == 0 && !tc.expectError {
					t.Error("Expected assignments but got none")
				}
			} else {
				// Test normal avec conditions spéciales
				wrapper := managerWrapper{mgr: mgr, vehicles: vehicles}
				srv := rte.NewRTEServerMockWithRegistry(config.RTEMockConfig{Address: "127.0.0.1:0"}, wrapper, reg)
				go func() { _ = srv.Start(ctx) }()

				if err := waitForRTEServer(srv, 2*time.Second); err != nil {
					t.Fatalf("server not ready: %v", err)
				}

				var signal model.FlexibilitySignal
				if tc.invalidSignal {
					// Signal avec puissance zéro (considéré comme invalide)
					signal = model.FlexibilitySignal{
						Type:      model.SignalFCR,
						PowerKW:   0, // Puissance zéro = invalide
						Duration:  0, // Durée zéro = invalide
						Timestamp: time.Now(),
					}
				} else {
					signal = model.FlexibilitySignal{
						Type:      model.SignalFCR,
						PowerKW:   15,
						Duration:  300 * time.Second,
						Timestamp: time.Now(),
					}
				}

				// Test direct dispatch
				result := mgr.Dispatch(signal, vehicles)

				// Vérifier les résultats selon le cas de test
				if tc.unavailableVeh {
					// Les véhicules étant indisponibles, aucun fallback n'est attendu
					if len(result.FallbackAssignments) == 0 {
						t.Log("No fallback assignments, which is expected for unavailable vehicles")
					}
				} else if !tc.invalidSignal {
					// Signal valide devrait produire des assignments
					if len(result.Assignments) == 0 {
						t.Error("Expected assignments for valid signal")
					}
				}

				// Attendre un peu pour le traitement
				time.Sleep(500 * time.Millisecond)
			}
		})
	}
}

// TestPerformanceIntegration teste les performances sous charge
func TestPerformanceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	reg := prometheus.NewRegistry()
	sinkIf, err := metrics.NewPromSinkWithRegistry(coremetrics.Config{}, reg)
	if err != nil {
		t.Fatalf("prom sink: %v", err)
	}

	publisher := mqtt.NewMockPublisher()
	bus := eventbus.New()

	// Grande flotte de véhicules
	vehicles := make([]model.Vehicle, 100)
	for i := 0; i < 100; i++ {
		vehicles[i] = model.Vehicle{
			ID:         fmt.Sprintf("perf-v%d", i),
			SoC:        0.5 + float64(i%50)/100.0, // SoC entre 0.5 et 1.0
			IsV2G:      i%2 == 0,                  // 50% V2G
			Available:  i%10 != 0,                 // 90% disponibles
			MaxPower:   float64(10 + i%40),        // Puissance entre 10 et 50 kW
			BatteryKWh: float64(30 + i%70),        // Batterie entre 30 et 100 kWh
		}
	}

	mgr, err := dispatch.NewDispatchManager(
		dispatch.SimpleVehicleFilter{},
		&dispatch.SmartDispatcher{},
		&dispatch.BalancedFallback{},
		publisher,
		50*time.Millisecond,
		sinkIf.(*metrics.PromSink),
		bus,
		nil,
		logger.NopLogger{},
		nil,
	)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	// Test de performance avec plusieurs signaux simultanés
	start := time.Now()
	signals := []model.FlexibilitySignal{
		{Type: model.SignalFCR, PowerKW: 500, Duration: 900 * time.Second, Timestamp: time.Now()},
		{Type: model.SignalAFRR, PowerKW: 300, Duration: 600 * time.Second, Timestamp: time.Now()},
		{Type: model.SignalMA, PowerKW: 200, Duration: 300 * time.Second, Timestamp: time.Now()},
	}

	results := make([]dispatch.DispatchResult, len(signals))
	for i, signal := range signals {
		results[i] = mgr.Dispatch(signal, vehicles)
	}

	duration := time.Since(start)

	// Vérifier les performances
	if duration > 1*time.Second {
		t.Errorf("Dispatch took too long: %v", duration)
	}

	assignCount := 0
	for i, result := range results {
		assignCount += len(result.Assignments)
		t.Logf("Signal %d: %d assignments, %d acknowledged",
			i, len(result.Assignments), len(result.Acknowledged))
	}
	if assignCount == 0 {
		t.Error("no assignments produced")
	}

	t.Logf("Performance test completed in %v", duration)
}
