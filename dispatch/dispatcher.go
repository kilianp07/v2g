package dispatch

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kilianp07/v2g/logger"
	mqttwrapper "github.com/kilianp07/v2g/mqtt"
)

// Dispatch sends orders based on the provided flexibility signal.
func (dm *DispatchManager) Dispatch(signal FlexibilitySignal) {
	selected, allocation, err := dm.SelectVehicles(signal)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("failed to select vehicles: %v", err))
		return
	}
	if len(selected) == 0 {
		logger.Log.Warn("No vehicles selected for this signal")
		return
	}

	var wg sync.WaitGroup
	for _, v := range selected {
		vehicle := v
		wg.Add(1)
		go func() {
			defer wg.Done()
			order := DispatchOrder{
				VehicleID: vehicle.ID,
				PowerKW:   allocation[vehicle.ID],
				Duration:  signal.Duration,
				Type:      signal.Type,
				Timestamp: time.Now(),
			}
			payload, err := json.Marshal(order)
			if err != nil {
				logger.Log.Error(fmt.Sprintf("marshal order for %s: %v", vehicle.ID, err))
				dm.HandleVehicleFeedback(vehicle.ID, false)
				return
			}
			topic := fmt.Sprintf("%s/%s", dm.topicPrefix, vehicle.ID)
			if err := dm.mqttClient.Publish(topic, payload, 1); err != nil {
				logger.Log.Error(fmt.Sprintf("failed to publish to %s: %v", vehicle.ID, err))
				dm.HandleVehicleFeedback(vehicle.ID, false)
				return
			}
			dm.HandleVehicleFeedback(vehicle.ID, true)
		}()
	}
	wg.Wait()
}

// SetMQTTClient sets the MQTT client for the dispatch manager.
func (dm *DispatchManager) SetMQTTClient(client mqttwrapper.MQTTClient, topic string) {
	dm.mqttClient = client
	dm.topicPrefix = topic
}
