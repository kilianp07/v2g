package metrics

import (
	"context"
	"strconv"
	"time"

	"github.com/kilianp07/v2g/dispatch/events"
	"github.com/kilianp07/v2g/internal/eventbus"
)

// StartEventCollector subscribes to the event bus and records metrics for events.
// It stops when the context is canceled.
func StartEventCollector(ctx context.Context, bus eventbus.EventBus, sink MetricsSink) {
	if bus == nil || sink == nil {
		return
	}
	sub := bus.Subscribe()
	go func() {
		defer bus.Unsubscribe(sub)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub:
				if !ok {
					return
				}
				switch e := ev.(type) {
				case events.SignalEvent:
					if r, ok := sink.(RTESignalRecorder); ok {
						_ = r.RecordRTESignal(RTESignalEvent{Signal: e.Signal, Time: time.Now()})
					}
				case events.AckEvent:
					if r, ok := sink.(DispatchAckRecorder); ok {
						errStr := ""
						if e.Err != nil {
							errStr = e.Err.Error()
						}
						_ = r.RecordDispatchAck(DispatchAckEvent{
							VehicleID:    e.VehicleID,
							Signal:       e.Signal,
							Acknowledged: e.Acknowledged,
							Latency:      e.Latency,
							Error:        errStr,
							DispatchID:   strconv.FormatInt(time.Now().UnixNano(), 10),
							Time:         time.Now(),
						})
					}
				}
			}
		}
	}()
}
