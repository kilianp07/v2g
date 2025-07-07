//go:build !no_containers

package test

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kilianp07/v2g/config"
	"github.com/kilianp07/v2g/test/util"
)

func runConfigTest(t *testing.T, cfgFile string) {
	ctx := context.Background()
	broker, cleanup, err := util.StartMosquitto(ctx)
	if err != nil {
		t.Fatalf("mosquitto: %v", err)
	}
	defer cleanup()

	dir := t.TempDir()
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	data = []byte(strings.ReplaceAll(string(data), "BROKER", broker))
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}

	bin := filepath.Join(dir, "svc")
	buildCmd := exec.Command("go", "build", "-o", bin, ".")
	buildCmd.Dir = ".."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	if err := os.Chmod(bin, 0o755); err != nil {
		t.Fatalf("chmod bin: %v", err)
	}

	cmd := exec.Command(bin, "--config", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start svc: %v", err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		req, _ := http.NewRequestWithContext(waitCtx, http.MethodGet, "http://"+cfg.RTE.Mock.Address+"/rte/ping", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_, err := io.Copy(io.Discard, resp.Body)
			if err != nil {
				t.Fatalf("read resp: %v", err)
			}
			err = resp.Body.Close()
			if err != nil {
				t.Fatalf("close resp: %v", err)
			}
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		select {
		case <-waitCtx.Done():
			t.Fatalf("server not ready: %v", waitCtx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}

	reqBody := `{"signal_type":"FCR","start_time":"2023-01-01T00:00:00Z","end_time":"2023-01-01T00:05:00Z","power":5}`
	resp, err := http.Post("http://"+cfg.RTE.Mock.Address+"/rte/signal", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestE2EConfig_EqualNoop(t *testing.T)     { runConfigTest(t, "configs/equal_noop.yaml") }
func TestE2EConfig_SmartBalanced(t *testing.T) { runConfigTest(t, "configs/smart_balanced.yaml") }
func TestE2EConfig_LPProb(t *testing.T)        { runConfigTest(t, "configs/lp_prob.yaml") }
