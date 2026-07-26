# FleetPulse PRD 2-4 Operations, Packaging, and Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete FleetPulse PRDs 2-4 with executable security/operations behavior, lifecycle packaging artifacts, and automated CI/release workflows.

**Architecture:** Keep the PRD 1 API stable and add operational layers around it: config parsing, token-file provisioning, bearer auth middleware, collector caching, diagnostics, CLI subcommands, packaging scripts, and GitHub Actions. Packaging and CI artifacts use the existing Go module and do not add runtime dependencies.

**Tech Stack:** Go 1.26 standard library, POSIX shell scripts, PowerShell scripts, systemd unit files, launchd plists, Dockerfile, Docker Compose, GitHub Actions YAML.

## Global Constraints

- Preserve the PRD 1 API schema and endpoints.
- Reject unauthenticated non-local network exposure.
- Store bearer tokens in protected files and never log them repeatedly.
- Keep upgrades token/config-preserving by default.
- Represent collector failures as partial/unavailable state, not total API failure.
- Provide deterministic lifecycle artifacts for Linux, macOS, Windows, and Docker.
- Provide CI and release workflows for tests, builds, artifacts, Docker validation, checksums, and traceability.

---

## File Structure

- `internal/config/config.go`: configuration defaults, file/env/flag merge helpers, validation.
- `internal/token/token.go`: token generation, provisioning, loading, rotation.
- `internal/api/server.go`: bearer auth middleware and health response.
- `internal/collector/collector.go`: cache TTL and health state.
- `cmd/fleetpulse/main.go`: subcommands and serve startup.
- `packaging/linux/*`: Linux install, uninstall, systemd unit.
- `packaging/macos/*`: macOS install, uninstall, launch daemon.
- `packaging/windows/*`: Windows install and uninstall scripts.
- `Dockerfile`: multi-stage container build.
- `deploy/docker-compose.yml`: persistent Docker deployment example.
- `.github/workflows/ci.yml`: PR/main validation.
- `.github/workflows/release.yml`: tagged release workflow.
- `docs/*.md`: operator documentation.

## Task 1: Config, Token, and Auth

- [ ] Write failing tests for config defaults, non-local auth validation, token provisioning permissions, token rotation, and unauthorized API responses.
- [ ] Run targeted tests and verify they fail because the new behavior does not exist.
- [ ] Implement `internal/config`, `internal/token`, and API bearer middleware.
- [ ] Run targeted tests and verify they pass.

## Task 2: Collector Cache, Health, and Diagnostics

- [ ] Write failing tests for snapshot cache reuse, cache refresh, collector health status, and diagnostics output that hides tokens.
- [ ] Run targeted tests and verify they fail because cache and diagnostics do not exist.
- [ ] Implement collector cache options, health reporting, and CLI `diagnose` / `config show` behavior.
- [ ] Run targeted tests and verify they pass.

## Task 3: CLI Serve and Token Commands

- [ ] Write failing CLI tests for `token show`, `token rotate`, `config show`, and safe serve validation.
- [ ] Run targeted tests and verify they fail because subcommands do not exist.
- [ ] Refactor `cmd/fleetpulse` into a testable `run(args, stdout, stderr)` command dispatcher.
- [ ] Run targeted tests and verify they pass.

## Task 4: Packaging and Operator Docs

- [ ] Add Linux, macOS, Windows, Docker, and Compose artifacts.
- [ ] Add install, uninstall, CLI, config, API, Docker, token, troubleshooting, permissions, disk health, upgrade, and rollback docs.
- [ ] Run shell syntax checks and PowerShell parser checks where available.

## Task 5: CI and Release

- [ ] Add CI workflow for formatting, tests, build matrix, Docker validation, and script syntax checks.
- [ ] Add release workflow for tagged builds, checksums, Docker image build, and GitHub release upload.
- [ ] Verify YAML parses with available local tooling.

## Task 6: Final Verification and Commit

- [ ] Run `gofmt`.
- [ ] Run `env GOCACHE=/private/tmp/fleetpulse-gocache GOMODCACHE=/private/tmp/fleetpulse-gomodcache go test ./...`.
- [ ] Run `env GOCACHE=/private/tmp/fleetpulse-gocache GOMODCACHE=/private/tmp/fleetpulse-gomodcache go build -o /private/tmp/fleetpulse ./cmd/fleetpulse`.
- [ ] Run packaging syntax checks.
- [ ] Commit the completed PRD2-4 implementation.

## Self-Review

- Spec coverage: PRD 2 executable security and operations, PRD 3 lifecycle artifacts and docs, and PRD 4 CI/release automation are covered.
- Placeholder scan: no TBD/TODO placeholders remain.
- Type consistency: config, token, API, collector, and CLI boundaries are named consistently.
