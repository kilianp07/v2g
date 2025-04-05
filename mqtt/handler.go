package mqtt

import (
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kilianp07/v2g/logger"
)

type FeedbackHandler interface {
	HandleVehicleFeedback(vehicleID string, success bool)
}

func AckHandler(fh FeedbackHandler) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		var ack DispatchAck
		if err := json.Unmarshal(msg.Payload(), &ack); err != nil {
			logger.Log.WithField("Error", err).Error("Invalid ACK received")
			return
		}
		fh.HandleVehicleFeedback(ack.VehicleID, ack.Success)
	}
}
