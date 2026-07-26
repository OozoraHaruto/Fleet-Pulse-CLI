package collector_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/haruto/fleetpulse/internal/collector"
	"github.com/haruto/fleetpulse/internal/schema"
)

func TestSnapshotIncludesStableIdentityAndPlatform(t *testing.T) {
	service := collector.NewService()

	snapshot := service.Snapshot(context.Background())

	if snapshot.SchemaVersion != "v1" {
		t.Fatalf("SchemaVersion = %q, want v1", snapshot.SchemaVersion)
	}
	if snapshot.Target.Hostname == "" {
		t.Fatal("Target.Hostname is empty")
	}
	if snapshot.Target.Platform != runtime.GOOS {
		t.Fatalf("Target.Platform = %q, want %q", snapshot.Target.Platform, runtime.GOOS)
	}
	if snapshot.Target.Architecture != runtime.GOARCH {
		t.Fatalf("Target.Architecture = %q, want %q", snapshot.Target.Architecture, runtime.GOARCH)
	}
	if snapshot.Timestamp.IsZero() {
		t.Fatal("Timestamp is zero")
	}
}

func TestSnapshotIncludesAvailableCPUAndUnsupportedGPU(t *testing.T) {
	service := collector.NewService()

	snapshot := service.Snapshot(context.Background())

	if snapshot.CPU.Status != schema.StatusAvailable {
		t.Fatalf("CPU.Status = %q, want %q", snapshot.CPU.Status, schema.StatusAvailable)
	}
	if snapshot.CPU.Scope != schema.ScopeHost {
		t.Fatalf("CPU.Scope = %q, want %q", snapshot.CPU.Scope, schema.ScopeHost)
	}
	if snapshot.CPU.CoreCount != runtime.NumCPU() {
		t.Fatalf("CPU.CoreCount = %d, want %d", snapshot.CPU.CoreCount, runtime.NumCPU())
	}
	if snapshot.GPU.Status != schema.StatusUnsupported {
		t.Fatalf("GPU.Status = %q, want %q", snapshot.GPU.Status, schema.StatusUnsupported)
	}
	if snapshot.GPU.Scope != schema.ScopeUnavailable {
		t.Fatalf("GPU.Scope = %q, want %q", snapshot.GPU.Scope, schema.ScopeUnavailable)
	}
}

func TestSnapshotKeepsUnavailableSectionsExplicit(t *testing.T) {
	service := collector.NewService()

	snapshot := service.Snapshot(context.Background())

	sections := map[string]schema.SectionStatus{
		"memory": snapshot.Memory.SectionStatus,
		"disks":  snapshot.Disks.SectionStatus,
		"system": snapshot.System.SectionStatus,
	}
	for name, section := range sections {
		if section.Status == "" {
			t.Fatalf("%s status is empty", name)
		}
		if section.Scope == "" {
			t.Fatalf("%s scope is empty", name)
		}
	}
}
