package schema_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/haruto/fleetpulse/internal/schema"
)

func TestSnapshotJSONDistinguishesZeroFromUnavailable(t *testing.T) {
	zero := 0.0
	s := schema.Snapshot{
		Timestamp:     time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC),
		SchemaVersion: "v1",
		Target: schema.TargetIdentity{
			Hostname:     "node-a",
			Platform:     "darwin",
			Architecture: "arm64",
		},
		CPU: schema.CPUSection{
			SectionStatus: schema.SectionStatus{
				Status: schema.StatusAvailable,
				Scope:  schema.ScopeHost,
			},
			CoreCount:   8,
			Utilization: &zero,
		},
		Memory: schema.MemorySection{
			SectionStatus: schema.SectionStatus{
				Status: schema.StatusAvailable,
				Scope:  schema.ScopeHost,
			},
			PercentUsed: &zero,
		},
		GPU: schema.GPUSection{
			SectionStatus: schema.SectionStatus{
				Status: schema.StatusUnsupported,
				Scope:  schema.ScopeUnavailable,
			},
		},
	}

	got, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	wantFragments := []string{
		`"schema_version":"v1"`,
		`"hostname":"node-a"`,
		`"utilization_percent":0`,
		`"percent_used":0`,
		`"gpu":{"status":"unsupported","scope":"unavailable"`,
	}
	for _, want := range wantFragments {
		if !strings.Contains(string(got), want) {
			t.Fatalf("snapshot JSON missing %s in %s", want, got)
		}
	}
}
