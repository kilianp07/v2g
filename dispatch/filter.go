package dispatch

import "github.com/kilianp07/v2g/model"

// SimpleVehicleFilter implements basic filtering rules based on the signal type.
type SimpleVehicleFilter struct{}

func (f SimpleVehicleFilter) Filter(vehicles []model.Vehicle, signal model.FlexibilitySignal) []model.Vehicle {
	var res []model.Vehicle
	for _, v := range vehicles {
		if !v.Available {
			continue
		}
		switch signal.Type {
		case model.SignalFCR:
			if v.IsV2G && v.SoC >= 0.6 {
				res = append(res, v)
			}
		case model.SignalNEBEF:
			if v.CanReduceCharge() {
				res = append(res, v)
			}
		default:
			res = append(res, v)
		}
	}
	return res
}
