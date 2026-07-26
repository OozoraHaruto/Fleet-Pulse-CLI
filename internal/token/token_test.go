package token_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haruto/fleetpulse/internal/token"
)

func TestEnsureCreatesProtectedTokenFileOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")

	first, created, err := token.EnsureFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("created = false, want true on first ensure")
	}
	if len(first) < 40 {
		t.Fatalf("token length = %d, want at least 40", len(first))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}

	second, created, err := token.EnsureFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("created = true, want false on second ensure")
	}
	if second != first {
		t.Fatal("EnsureFile changed an existing token")
	}
}

func TestRotateReplacesToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	first, _, err := token.EnsureFile(path)
	if err != nil {
		t.Fatal(err)
	}

	second, err := token.RotateFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Fatal("RotateFile returned the previous token")
	}
	loaded, err := token.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != second {
		t.Fatal("LoadFile did not return the rotated token")
	}
}
