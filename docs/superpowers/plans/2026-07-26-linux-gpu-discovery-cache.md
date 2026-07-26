# Linux GPU Discovery Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Linux GPU discovery, persistent discovery caching, and best-effort vendor collection across Raspberry Pi, NVIDIA, AMD, Intel, and generic DRM/sysfs.

**Architecture:** Linux `collectGPU` will delegate to a small collector object with injected filesystem, command, clock, and hostname dependencies. Discovery results are persisted as JSON under `/var/lib/fleetpulse/gpu-discovery.json`, reused while valid, and refreshed on staleness, fingerprint mismatch, or repeated cached collection failure. Platform-neutral parsing lives in testable helpers; Linux-only syscall and command wiring stays in `collector_linux.go`.

**Tech Stack:** Go standard library only; JSON cache file; Linux `/proc`, `/sys/class/drm`, optional `nvidia-smi`; existing `schema.GPUSection`.

## Global Constraints

- Linux only; macOS keeps the existing Darwin GPU collector.
- Do not require privileged access.
- Do not install vendor packages.
- Do not change the v1 JSON schema.
- Missing cache, invalid cache, and cache write failures must not crash collection.
- Available discovery TTL is 24 hours.
- Unsupported discovery TTL is 1 hour.
- Cached collection rediscovery threshold is 2 consecutive failures.
- Tests must be written before production code and verified red/green.

---

### Task 1: Linux GPU Cache And Orchestrator

**Files:**
- Create: `internal/collector/collector_linux_gpu.go`
- Create: `internal/collector/collector_linux_gpu_test.go`
- Modify: `internal/collector/collector_linux.go`

**Interfaces:**
- Consumes: `schema.GPUSection`, `unsupported(reason string) schema.SectionStatus`, `unavailable(reason string) schema.SectionStatus`.
- Produces: `newLinuxGPUCollector() linuxGPUCollector`, `func (c linuxGPUCollector) collect(ctx context.Context) schema.GPUSection`, cache read/write helpers, detector registry behavior.

- [x] **Step 1: Write failing tests**

Add tests proving fresh cached discovery collects without rediscovering, stale cache rediscoveries, cached collection failure increments failure count, threshold failure rediscoveries, unsupported cache expires sooner, invalid cache is ignored, and cache write failures do not fail live collection.

- [x] **Step 2: Verify tests fail**

Run: `GOCACHE=/private/tmp/fleetpulse-go-build go test ./internal/collector -run 'TestLinuxGPU'`
Expected: FAIL because Linux GPU cache types and helpers do not exist.

- [x] **Step 3: Implement orchestrator**

Add injected dependencies for cache path, now, hostname, read/write file, mkdir, command lookup/run, and sysfs root. Add cache JSON structs and TTL/failure policy. Update Linux `collectGPU` to call `newLinuxGPUCollector().collect(ctx)`.

- [x] **Step 4: Verify tests pass**

Run: `GOCACHE=/private/tmp/fleetpulse-go-build go test ./internal/collector -run 'TestLinuxGPU'`
Expected: PASS.

### Task 2: Linux GPU Detector Parsers

**Files:**
- Modify: `internal/collector/collector_linux_gpu.go`
- Modify: `internal/collector/collector_linux_gpu_test.go`

**Interfaces:**
- Consumes: orchestrator detector interface from Task 1.
- Produces: Raspberry Pi, NVIDIA, AMD, Intel, and DRM detector discovery/collection helpers.

- [x] **Step 1: Write failing parser tests**

Add tests for Raspberry Pi model parsing, NVIDIA CSV parsing, AMD and Intel sysfs vendor discovery, DRM fallback identity, metric parsing for readable sysfs fields, and detector priority order.

- [x] **Step 2: Verify tests fail**

Run: `GOCACHE=/private/tmp/fleetpulse-go-build go test ./internal/collector -run 'TestLinuxGPU'`
Expected: FAIL because detector parsers are not implemented.

- [x] **Step 3: Implement detectors**

Implement Raspberry Pi model/DRM identity, NVIDIA `nvidia-smi` discovery and CSV collection, AMD/Intel DRM sysfs discovery with optional readable metrics, and generic DRM fallback identity.

- [x] **Step 4: Verify tests pass**

Run: `GOCACHE=/private/tmp/fleetpulse-go-build go test ./internal/collector -run 'TestLinuxGPU'`
Expected: PASS.

### Task 3: Documentation And Full Verification

**Files:**
- Modify: `README.md`
- Modify: `docs/API_REFERENCE.md`
- Modify: `docs/TROUBLESHOOTING.md`
- Modify: `docs/DISK_HEALTH.md`
- Modify: `docs/superpowers/plans/2026-07-26-linux-gpu-discovery-cache.md`

**Interfaces:**
- Consumes: completed Linux GPU behavior.
- Produces: operator-facing docs and checked plan boxes.

- [x] **Step 1: Update docs**

Document Linux GPU discovery caching, vendor-dependent metrics, cache deletion for rediscovery, and unsupported versus unavailable meanings.

- [x] **Step 2: Run full verification**

Run:

```sh
GOCACHE=/private/tmp/fleetpulse-go-build GOMODCACHE=/private/tmp/fleetpulse-go-mod go test -count=1 ./...
GOCACHE=/private/tmp/fleetpulse-go-build GOMODCACHE=/private/tmp/fleetpulse-go-mod go test -race -count=1 ./...
GOCACHE=/private/tmp/fleetpulse-go-build GOMODCACHE=/private/tmp/fleetpulse-go-mod go vet ./...
GOCACHE=/private/tmp/fleetpulse-go-build GOMODCACHE=/private/tmp/fleetpulse-go-mod GOOS=linux GOARCH=arm64 go test -c ./internal/collector -o /private/tmp/fleetpulse-collector-linux-arm64.test
GOCACHE=/private/tmp/fleetpulse-go-build GOMODCACHE=/private/tmp/fleetpulse-go-mod GOOS=linux GOARCH=arm64 go build -o /private/tmp/fleetpulse-linux-arm64 ./cmd/fleetpulse
```

Expected: all commands pass.

- [x] **Step 3: Commit**

Stage all Linux collector, GPU cache, docs, and plan changes. Commit with:

```sh
git commit -m "feat: add linux gpu discovery cache"
```
