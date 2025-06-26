package main

import (
	"math"
	"sync"
	"time"
)

// Battery models a simple EV battery with charge/discharge limits.
type Battery struct {
	CapacityKWh     float64 // total capacity
	Soc             float64 // state of charge [0,1]
	ChargeRateKW    float64 // maximum charging power
	DischargeRateKW float64 // maximum discharging power
	mu              sync.Mutex
}

// ApplyPower updates the SoC according to the requested power and duration.
// Positive power means discharge (injection), negative means charging.
// It returns the actual power applied after enforcing limits.
func (b *Battery) ApplyPower(powerKW float64, dt time.Duration) float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	hours := dt.Hours()
	if hours <= 0 {
		return 0
	}

	actual := powerKW
	if powerKW > 0 { // discharge
		if powerKW > b.DischargeRateKW {
			actual = b.DischargeRateKW
		}
		maxEnergy := b.Soc * b.CapacityKWh
		needed := actual * hours
		if needed > maxEnergy {
			needed = maxEnergy
			if hours > 0 {
				actual = needed / hours
			}
		}
		b.Soc -= needed / b.CapacityKWh
	} else if powerKW < 0 { // charge
		p := math.Abs(powerKW)
		if p > b.ChargeRateKW {
			p = b.ChargeRateKW
		}
		avail := (1 - b.Soc) * b.CapacityKWh
		needed := p * hours
		if needed > avail {
			needed = avail
			if hours > 0 {
				p = needed / hours
			}
		}
		b.Soc += needed / b.CapacityKWh
		actual = -p
	}

	if b.Soc < 0 {
		b.Soc = 0
	}
	if b.Soc > 1 {
		b.Soc = 1
	}
	return actual
}
