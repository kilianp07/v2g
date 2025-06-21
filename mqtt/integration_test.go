package mqttwrapper

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestIntegration verifies publishing and subscribing using a real Mosquitto broker.
func TestIntegration(t *testing.T) {
	if os.Getenv("DOCKER_AVAILABLE") != "true" && os.Getenv("DOCKER_AVAILABLE") != "1" {
		t.Skip("docker not available")
	}
	ctx := context.Background()
	req := tc.ContainerRequest{
		Image:        "eclipse-mosquitto:2.0",
		ExposedPorts: []string{"1883/tcp"},
		WaitingFor:   wait.ForListeningPort("1883/tcp"),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	}()

	// give broker time to fully start
	time.Sleep(500 * time.Millisecond)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "1883")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	brokerURL := fmt.Sprintf("tcp://%s:%s", host, port.Port())
	client := &MQTTClientWrapper{}
	var connectErr error
	for i := 0; i < 5; i++ {
		connectErr = client.Connect(brokerURL, "pub")
		if connectErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if connectErr != nil {
		t.Fatalf("failed to connect: %v", connectErr)
	}
	defer client.Disconnect()

	msgCh := make(chan string, 1)
	if err := client.Subscribe("test/topic", 0, func(c MQTTClient, m Message) {
		msgCh <- string(m.Payload())
	}); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	payload := "hello"
	if err := client.Publish("test/topic", payload, 0); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	select {
	case got := <-msgCh:
		if got != payload {
			t.Fatalf("expected %s got %s", payload, got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}
