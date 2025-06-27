package model

import "testing"

func TestVehicleEffectiveCapacity(t *testing.T) {
	v := Vehicle{MaxPower: 50, SoC: 1}
	cap := v.EffectiveCapacity(30)
	if cap != 20 {
		t.Fatalf("expected 20 got %v", cap)
	}
}

func TestVehicleEffectiveCapacityWithParams(t *testing.T) {
	v := Vehicle{MaxPower: 50, SoC: 0.8, AvailabilityProb: 0.5, DegradationFactor: 0.2}
	// Max usable power: 50*(1-0.2)=40, minus current 10 =>30. Weighted by SoC and availability: 30*0.8*0.5=12
	cap := v.EffectiveCapacity(10)
	if cap != 12 {
		t.Fatalf("expected 12 got %v", cap)
	}
}

func TestVehicleEffectiveCapacityLowSoC(t *testing.T) {
	v := Vehicle{MaxPower: 50, SoC: 0.1, AvailabilityProb: 1}
	cap := v.EffectiveCapacity(0)
	if cap != 0 {
		t.Fatalf("expected 0 got %v", cap)
	}
}
