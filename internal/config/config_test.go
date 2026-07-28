package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/haruto/fleetpulse/internal/config"
)

func TestDefaultConfigIsNetworkBoundAndAuthenticated(t *testing.T) {
	cfg := config.Default()

	if cfg.Addr != "0.0.0.0:35338" {
		t.Fatalf("Addr = %q, want 0.0.0.0:35338", cfg.Addr)
	}
	if !cfg.AuthEnabled {
		t.Fatal("AuthEnabled = false, want true for network default")
	}
	if cfg.TokenFile == "" {
		t.Fatal("TokenFile is empty")
	}
	if !strings.Contains(filepath.ToSlash(cfg.TokenFile), "fleetpulse") {
		t.Fatalf("TokenFile = %q, want path containing fleetpulse", cfg.TokenFile)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected default config: %v", err)
	}
	if cfg.CacheTTL != 10*time.Second {
		t.Fatalf("CacheTTL = %s, want 10s", cfg.CacheTTL)
	}
	if cfg.CollectorTimeout != 2*time.Second {
		t.Fatalf("CollectorTimeout = %s, want 2s", cfg.CollectorTimeout)
	}
}

func TestValidateRejectsNonLocalBindWithoutAuth(t *testing.T) {
	cfg := config.Default()
	cfg.Addr = "0.0.0.0:8080"
	cfg.AuthEnabled = false

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate allowed non-local unauthenticated bind")
	}
}

func TestValidateAllowsNonLocalBindWithAuth(t *testing.T) {
	cfg := config.Default()
	cfg.Addr = "0.0.0.0:8080"
	cfg.AuthEnabled = true

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected authenticated non-local bind: %v", err)
	}
}

func TestLoadFileParsesDurationsAndCollectors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fleetpulse.json")
	body := []byte(`{
		"addr":"0.0.0.0:9090",
		"auth_enabled":true,
		"token_file":"/state/token",
		"cache_ttl":"30s",
		"collector_timeout":"1500ms",
		"enabled_collectors":["cpu","memory"],
		"log_level":"debug",
		"service_name":"fleetpulse-dev",
		"deployment_target":"docker"
	}`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Addr != "0.0.0.0:9090" {
		t.Fatalf("Addr = %q, want 0.0.0.0:9090", cfg.Addr)
	}
	if cfg.CacheTTL != 30*time.Second {
		t.Fatalf("CacheTTL = %s, want 30s", cfg.CacheTTL)
	}
	if cfg.CollectorTimeout != 1500*time.Millisecond {
		t.Fatalf("CollectorTimeout = %s, want 1500ms", cfg.CollectorTimeout)
	}
	if got := cfg.EnabledCollectors; len(got) != 2 || got[0] != "cpu" || got[1] != "memory" {
		t.Fatalf("EnabledCollectors = %#v, want cpu,memory", got)
	}
}
