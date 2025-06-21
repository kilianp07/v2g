package dispatch

import "github.com/kilianp07/v2g/model"

// EqualDispatcher distributes power equally between all vehicles.
type EqualDispatcher struct{}

func (d EqualDispatcher) Dispatch(vehicles []model.Vehicle, signal model.FlexibilitySignal) map[string]float64 {
	assignments := make(map[string]float64)
	if len(vehicles) == 0 {
		return assignments
	}

	share := signal.PowerKW / float64(len(vehicles))
	remaining := signal.PowerKW
	var needMore []model.Vehicle
	for _, v := range vehicles {
		amount := share
		if v.MaxPower < amount {
			amount = v.MaxPower
		} else {
			needMore = append(needMore, v)
		}
		assignments[v.ID] = amount
		remaining -= amount
	}

	for remaining > 0 && len(needMore) > 0 {
		share = remaining / float64(len(needMore))
		next := needMore[:0]
		for _, v := range needMore {
			cap := v.MaxPower - assignments[v.ID]
			if cap <= 0 {
				continue
			}
			add := share
			if cap < add {
				add = cap
			}
			assignments[v.ID] += add
			remaining -= add
			if assignments[v.ID] < v.MaxPower && remaining > 0 {
				next = append(next, v)
			}
		}
		if len(next) == len(needMore) {
			break
		}
		needMore = next
	}

	return assignments
}
