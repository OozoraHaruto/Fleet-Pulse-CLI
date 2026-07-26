package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/haruto/fleetpulse/internal/token"
)

func TestRunTokenShowPrintsExistingToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	want, _, err := token.EnsureFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"token", "show", "-token-file", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != want {
		t.Fatalf("stdout = %q, want token", stdout.String())
	}
}

func TestRunTokenRotateReplacesToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	first, _, err := token.EnsureFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"token", "rotate", "-token-file", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	second := strings.TrimSpace(stdout.String())
	if second == "" || second == first {
		t.Fatalf("rotated token = %q, first = %q", second, first)
	}
}

func TestRunConfigShowIncludesSafeDefaults(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"config", "show"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"addr": "127.0.0.1:8080"`) {
		t.Fatalf("config output missing default addr: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"auth_enabled": false`) {
		t.Fatalf("config output missing auth default: %s", stdout.String())
	}
}

func TestRunDiagnoseDoesNotLeakToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	secret, _, err := token.EnsureFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"diagnose", "-token-file", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), secret) {
		t.Fatalf("diagnostics leaked token: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"token_file_present": true`) {
		t.Fatalf("diagnostics missing token file presence: %s", stdout.String())
	}
}

func TestRunServeRejectsNonLocalBindWithoutAuth(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"serve", "-addr", "0.0.0.0:8080", "-auth=false"}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("exit code = 0, want validation failure")
	}
	if !strings.Contains(stderr.String(), "authentication is required") {
		t.Fatalf("stderr missing validation error: %s", stderr.String())
	}
}

func TestRunConfigShowLoadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fleetpulse.json")
	if err := os.WriteFile(path, []byte(`{"addr":"127.0.0.1:9090","auth_enabled":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"config", "show", "-config", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"addr": "127.0.0.1:9090"`) {
		t.Fatalf("config output missing file addr: %s", stdout.String())
	}
}
