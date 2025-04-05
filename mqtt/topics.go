package mqtt

func DispatchTopic(vehicleID string) string {
	return "v2g/dispatch/" + vehicleID
}

func AckTopic(vehicleID string) string {
	return "v2g/ack/" + vehicleID
}
