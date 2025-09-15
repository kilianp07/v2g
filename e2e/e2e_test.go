package e2e

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// junitReport is a minimal representation of a JUnit XML report. The E2E
// suite writes such a report so CI systems can display the results.
type junitReport struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name    string  `xml:"name,attr"`
	Failure *string `xml:"failure,omitempty"`
	Time    float64 `xml:"time,attr"`
}

// writeJUnit writes the provided report to the given path.
func writeJUnit(path string, rep junitReport) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	return enc.Encode(rep)
}

// startInflux starts an InfluxDB 2.7 container and returns it along with the
// base URL. The container is left running until the context is cancelled.
func startInflux(ctx context.Context, t *testing.T) (tc.Container, string) {
	t.Helper()
	req := tc.ContainerRequest{
		Image:        "influxdb:2.7",
		ExposedPorts: []string{"8086/tcp"},
		WaitingFor:   wait.ForHTTP("/health").WithPort("8086/tcp").WithStartupTimeout(60 * time.Second),
	}
	cont, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Skipf("unable to start influx container: %v", err)
	}
	host, _ := cont.Host(ctx)
	port, _ := cont.MappedPort(ctx, "8086")
	url := fmt.Sprintf("http://%s:%s", host, port.Port())
	return cont, url
}

// startMosquitto spins up a basic Mosquitto broker for tests.
func startMosquitto(ctx context.Context, t *testing.T) (tc.Container, string) {
	t.Helper()
	req := tc.ContainerRequest{
		Image:        "eclipse-mosquitto:2.0",
		ExposedPorts: []string{"1883/tcp"},
		WaitingFor:   wait.ForListeningPort("1883/tcp"),
	}
	cont, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Skipf("unable to start mosquitto: %v", err)
	}
	host, _ := cont.Host(ctx)
	port, _ := cont.MappedPort(ctx, "1883")
	return cont, fmt.Sprintf("tcp://%s:%s", host, port.Port())
}

// Test_E2E_DemoAssurance is a lightweight smoke test ensuring that the basic
// infrastructure (InfluxDB + Mosquitto) can be orchestrated via
// testcontainers-go. It does **not** run the full dispatch flow yet but serves
// as a placeholder for the demo assurance suite.
func Test_E2E_DemoAssurance(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skipf("docker not installed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	influxCont, influxURL := startInflux(ctx, t)
	if influxCont != nil {
		defer influxCont.Terminate(ctx) //nolint:errcheck
	}
	mqttCont, mqttURL := startMosquitto(ctx, t)
	if mqttCont != nil {
		defer mqttCont.Terminate(ctx) //nolint:errcheck
	}
	t.Logf("InfluxDB started at %s", influxURL)
	t.Logf("Mosquitto started at %s", mqttURL)

	// Set up Influx bucket
	org := "e2e_org"
	bucket := "e2e_bucket"
	token := "e2e-token"
	cli := NewInfluxClient(influxURL, org, bucket, token)
	defer cli.Close()
	if err := cli.SetupBucket(ctx); err != nil {
		t.Fatalf("setup bucket: %v", err)
	}

	if err := cli.WritePoint(ctx, "demo", nil, map[string]interface{}{"value": 1}, time.Now()); err != nil {
		t.Fatalf("write point: %v", err)
	}

	res, err := cli.Query(ctx, fmt.Sprintf(`from(bucket:"%s") |> range(start:-1m)`, bucket))
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer res.Close()
	count := 0
	for res.Next() {
		count++
	}
	if count == 0 {
		t.Fatalf("no points returned from Influx")
	}
	t.Logf("Influx query returned %d points", count)

	// Produce JUnit report
	dir := t.TempDir()
	rep := junitReport{Name: "e2e", Tests: 1, Cases: []junitTestCase{{Name: "Test_E2E_DemoAssurance", Time: 0}}}
	if err := writeJUnit(filepath.Join(dir, "e2e.xml"), rep); err != nil {
		t.Logf("write junit: %v", err)
	}
}
