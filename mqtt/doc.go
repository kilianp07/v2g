// Package mqtt provides an interface for sending power commands to vehicles
// over MQTT and receiving acknowledgments. It includes an implementation based
// on the Eclipse Paho client and a mock publisher used in tests. A fleet
// discovery component is also available to broadcast a request and collect
// vehicle states via MQTT.
package mqtt
