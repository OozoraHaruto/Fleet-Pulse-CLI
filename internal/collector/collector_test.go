package collector_test

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/haruto/fleetpulse/internal/collector"
	"github.com/haruto/fleetpulse/internal/schema"
)

var errCollectionFailed = errors.New("collection failed")

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

func TestSnapshotIncludesAvailableCPUAndExplicitGPUStatus(t *testing.T) {
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
	if snapshot.GPU.Status == "" {
		t.Fatal("GPU.Status is empty")
	}
	if snapshot.GPU.Scope == "" {
		t.Fatal("GPU.Scope is empty")
	}
	if snapshot.GPU.Status == schema.StatusAvailable && len(snapshot.GPU.Devices) == 0 {
		t.Fatal("GPU.Devices is empty for available GPU section")
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

func TestSnapshotReusesCacheInsideTTL(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	calls := 0
	service := collector.NewServiceWithOptions(collector.Options{
		Now:      func() time.Time { return now },
		CacheTTL: time.Minute,
		Collect: func(context.Context) (schema.Snapshot, error) {
			calls++
			return schema.Snapshot{
				Timestamp:     now,
				SchemaVersion: "v1",
				Target:        schema.TargetIdentity{Hostname: "node-a"},
			}, nil
		},
	})

	first := service.Snapshot(context.Background())
	second := service.Snapshot(context.Background())

	if calls != 1 {
		t.Fatalf("collect calls = %d, want 1", calls)
	}
	if !first.Timestamp.Equal(second.Timestamp) {
		t.Fatalf("cached timestamp changed from %s to %s", first.Timestamp, second.Timestamp)
	}
}

func TestSnapshotRefreshesCacheAfterTTL(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	calls := 0
	service := collector.NewServiceWithOptions(collector.Options{
		Now:      func() time.Time { return now },
		CacheTTL: time.Minute,
		Collect: func(context.Context) (schema.Snapshot, error) {
			calls++
			return schema.Snapshot{
				Timestamp:     now,
				SchemaVersion: "v1",
				Target:        schema.TargetIdentity{Hostname: "node-a"},
			}, nil
		},
	})

	service.Snapshot(context.Background())
	now = now.Add(time.Minute + time.Second)
	service.Snapshot(context.Background())

	if calls != 2 {
		t.Fatalf("collect calls = %d, want 2", calls)
	}
}

func TestSnapshotKeepsLastSuccessWhenRefreshFails(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	calls := 0
	service := collector.NewServiceWithOptions(collector.Options{
		Now:      func() time.Time { return now },
		CacheTTL: time.Second,
		Collect: func(context.Context) (schema.Snapshot, error) {
			calls++
			if calls == 2 {
				return schema.Snapshot{}, errCollectionFailed
			}
			return schema.Snapshot{
				Timestamp:     now,
				SchemaVersion: "v1",
				Target:        schema.TargetIdentity{Hostname: "node-a"},
			}, nil
		},
	})

	first := service.Snapshot(context.Background())
	now = now.Add(2 * time.Second)
	second := service.Snapshot(context.Background())
	health := service.Health(context.Background())

	if !second.Timestamp.Equal(first.Timestamp) {
		t.Fatalf("snapshot timestamp = %s, want cached %s", second.Timestamp, first.Timestamp)
	}
	if health.CollectionStatus != "degraded" {
		t.Fatalf("CollectionStatus = %q, want degraded", health.CollectionStatus)
	}
	if health.LastError == "" {
		t.Fatal("LastError is empty")
	}
}
