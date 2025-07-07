package metrics

import (
	core "github.com/kilianp07/v2g/core/metrics"
	eco "github.com/kilianp07/v2g/core/metrics/eco"
	"github.com/prometheus/client_golang/prometheus"
)

// EcoSink records dispatch results as ecological KPIs.
type EcoSink struct {
	store    eco.Store
	factor   float64
	injected *prometheus.GaugeVec
	ratio    *prometheus.GaugeVec
	co2      *prometheus.GaugeVec
}

// NewEcoSink creates a sink with Prometheus gauges registered on reg.
func NewEcoSink(store eco.Store, factor float64, reg prometheus.Registerer) *EcoSink {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	inj := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "vehicle_injected_energy_kwh",
		Help: "Daily injected energy per vehicle",
	}, []string{"vehicle_id", "day"})
	ratio := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "vehicle_energy_ratio",
		Help: "Daily ratio of injected to consumed energy",
	}, []string{"vehicle_id", "day"})
	co2 := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "vehicle_co2_avoided_grams",
		Help: "Daily CO2 avoided per vehicle",
	}, []string{"vehicle_id", "day"})
	reg.MustRegister(inj, ratio, co2)
	return &EcoSink{store: store, factor: factor, injected: inj, ratio: ratio, co2: co2}
}

// RecordDispatchResult processes dispatch results to update KPIs.
func (s *EcoSink) RecordDispatchResult(res []core.DispatchResult) error {
	for _, r := range res {
		kwh := r.PowerKW * r.Signal.Duration.Hours()
		rec := eco.Record{VehicleID: r.VehicleID, Date: r.Signal.Timestamp}
		if r.PowerKW >= 0 {
			rec.InjectedKWh = kwh
		} else {
			rec.ConsumedKWh = -kwh
		}
		if err := s.store.Add(rec); err != nil {
			return err
		}
		dayStr := eco.Day(rec.Date).Format("2006-01-02")
		records, _ := s.store.Query(r.VehicleID, rec.Date, rec.Date)
		if len(records) > 0 {
			rr := records[0]
			s.injected.WithLabelValues(r.VehicleID, dayStr).Set(rr.InjectedKWh)
			s.ratio.WithLabelValues(r.VehicleID, dayStr).Set(rr.EnergyRatio())
			s.co2.WithLabelValues(r.VehicleID, dayStr).Set(rr.CO2Avoided(s.factor))
		}
	}
	return nil
}
