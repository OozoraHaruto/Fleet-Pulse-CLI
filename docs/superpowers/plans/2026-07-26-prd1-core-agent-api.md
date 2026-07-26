# FleetPulse PRD 1 Core Agent and API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first FleetPulse Go agent slice: normalized telemetry schema, collector orchestration, and read-only HTTP API endpoints from PRD 1.

**Architecture:** A single Go binary starts an HTTP server backed by a collector service. The API layer formats stable v1 JSON responses while the collector layer returns explicit status and scope metadata for available, unsupported, and unavailable metrics.

**Tech Stack:** Go 1.26, standard library `net/http`, `encoding/json`, `testing`, and platform-specific standard library files where needed.

## Global Constraints

- Implement PRD 1 only: no authentication, token lifecycle, installers, Docker image, CI, or release automation.
- Expose `GET /health`, `GET /v1/stats`, `GET /v1/cpu`, `GET /v1/memory`, `GET /v1/disks`, `GET /v1/gpu`, and `GET /v1/system`.
- JSON responses include `timestamp`, target identity, platform information, and explicit metric availability.
- Unsupported metrics must be represented explicitly instead of failing the whole response.
- Keep the schema stable for later PRDs.
- Use TDD for production behavior.

---

## File Structure

- `go.mod`: module declaration.
- `cmd/fleetpulse/main.go`: parses `-addr`, starts the HTTP server.
- `internal/schema/schema.go`: stable JSON response structs and status constants.
- `internal/collector/collector.go`: snapshot service and collector implementations.
- `internal/api/server.go`: HTTP router and handlers.
- `internal/api/server_test.go`: API contract and route tests.
- `internal/collector/collector_test.go`: collector orchestration tests.

### Task 1: Go Module and Schema Contract

**Files:**
- Create: `go.mod`
- Create: `internal/schema/schema.go`
- Test: `internal/schema/schema_test.go`

**Interfaces:**
- Produces: `schema.Status`, `schema.Scope`, `schema.Snapshot`, and section structs used by collectors and API handlers.

- [ ] **Step 1: Write the failing schema JSON test**

```go
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
		Target:        schema.TargetIdentity{Hostname: "node-a", Platform: "darwin", Architecture: "arm64"},
		CPU: schema.CPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			CoreCount:     8,
			Utilization:   &zero,
		},
		GPU: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusUnsupported, Scope: schema.ScopeUnavailable},
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
		`"gpu":{"status":"unsupported","scope":"unavailable"`,
	}
	for _, want := range wantFragments {
		if !strings.Contains(string(got), want) {
			t.Fatalf("snapshot JSON missing %s in %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/schema -run TestSnapshotJSONDistinguishesZeroFromUnavailable -v`
Expected: FAIL because the module and schema package do not exist.

- [ ] **Step 3: Implement schema structs**

Add `go.mod` with module `github.com/haruto/fleetpulse`. Add typed status/scope constants, `SectionStatus`, `TargetIdentity`, `Snapshot`, and metric section structs with the JSON field names used by the PRD.

- [ ] **Step 4: Run schema tests**

Run: `go test ./internal/schema -v`
Expected: PASS.

### Task 2: Collector Snapshot Service

**Files:**
- Create: `internal/collector/collector.go`
- Test: `internal/collector/collector_test.go`

**Interfaces:**
- Consumes: `schema.Snapshot` and section structs.
- Produces: `collector.Service` with `Snapshot(context.Context) schema.Snapshot`, `CPU(context.Context) schema.CPUSection`, `Memory(context.Context) schema.MemorySection`, `Disks(context.Context) schema.DisksSection`, `GPU(context.Context) schema.GPUSection`, and `System(context.Context) schema.SystemSection`.

- [ ] **Step 1: Write failing collector tests**

Write tests proving `Snapshot` includes `schema_version`, target hostname, platform, architecture, CPU core count, and unsupported GPU status.

- [ ] **Step 2: Run collector tests to verify failure**

Run: `go test ./internal/collector -v`
Expected: FAIL because the collector package does not exist.

- [ ] **Step 3: Implement minimal collectors**

Use `os.Hostname`, `runtime.GOOS`, `runtime.GOARCH`, and `runtime.NumCPU`. Return memory, disks, uptime, load, and GPU with explicit `unsupported` or `unavailable` statuses when values cannot be collected with the standard library.

- [ ] **Step 4: Run collector tests**

Run: `go test ./internal/collector -v`
Expected: PASS.

### Task 3: HTTP API

**Files:**
- Create: `internal/api/server.go`
- Test: `internal/api/server_test.go`

**Interfaces:**
- Consumes: collector methods listed in Task 2.
- Produces: `api.NewHandler(service interface) http.Handler`.

- [ ] **Step 1: Write failing API tests**

Write tests for `/health`, `/v1/stats`, `/v1/cpu`, `/v1/memory`, `/v1/disks`, `/v1/gpu`, `/v1/system`, unknown routes, and unsupported methods.

- [ ] **Step 2: Run API tests to verify failure**

Run: `go test ./internal/api -v`
Expected: FAIL because the API package does not exist.

- [ ] **Step 3: Implement router and handlers**

Use `http.ServeMux`, encode all responses as JSON, set `Content-Type: application/json`, return 404/405 as JSON errors, and delegate section endpoints to the collector service.

- [ ] **Step 4: Run API tests**

Run: `go test ./internal/api -v`
Expected: PASS.

### Task 4: CLI Entrypoint and Full Verification

**Files:**
- Create: `cmd/fleetpulse/main.go`

**Interfaces:**
- Consumes: `api.NewHandler` and `collector.NewService`.
- Produces: runnable binary with `-addr` flag defaulting to `127.0.0.1:8080`.

- [ ] **Step 1: Implement entrypoint**

Parse `-addr`, construct the collector service and API handler, then start `http.ListenAndServe`.

- [ ] **Step 2: Run full tests**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 3: Build binary**

Run: `go build ./cmd/fleetpulse`
Expected: PASS.

---

## Self-Review

- Spec coverage: PRD 1 API endpoints, normalized schema, platform identity, explicit unsupported/unavailable states, and partial snapshot behavior are covered.
- Placeholder scan: no TBD/TODO placeholders remain.
- Type consistency: schema structs are consumed by collector and API tasks with stable names.
