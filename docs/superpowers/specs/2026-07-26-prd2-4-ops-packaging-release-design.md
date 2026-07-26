# FleetPulse PRD 2-4 Operations, Packaging, and Release Design

## Scope

This design completes the remaining PRD set at a repository-ready level. PRD 2 becomes executable security and operations behavior in the Go agent. PRD 3 becomes deterministic lifecycle scripts, service definitions, Docker packaging, and operator documentation. PRD 4 becomes automated tests, GitHub Actions workflows, release packaging, checksums, and release documentation.

## Approach

FleetPulse keeps the PRD 1 API contract and adds operational layers around it. Configuration is loaded from safe defaults, an optional JSON config file, environment variables, and command-line flags. Authentication uses a static bearer token stored in a protected file. Non-local binds are rejected unless authentication is enabled. Collector results are cached for a configurable TTL so a temporary collector failure does not break the full API response.

## PRD 2 Components

- `internal/config`: defaults, JSON config loading, environment overrides, bind/auth validation.
- `internal/token`: token generation, protected file provisioning, show, and rotate.
- `internal/api`: bearer-token middleware and richer health output.
- `internal/collector`: cache TTL and collection health metadata.
- `cmd/fleetpulse`: subcommands for `serve`, `token show`, `token rotate`, `diagnose`, and `config show`.

## PRD 3 Components

- Linux shell installer, uninstaller, and systemd unit template.
- macOS shell installer, uninstaller, and launch daemon plist.
- Windows PowerShell installer and uninstaller for Windows service registration.
- Dockerfile and Compose example with volume-backed `/var/lib/fleetpulse` state.
- Operator docs covering install, uninstall, CLI, config, API, Docker, tokens, troubleshooting, permissions, disk health, upgrade, and rollback.

The lifecycle artifacts are deterministic and preserve token/config state by default. Removal scripts accept explicit purge-style options for deleting preserved state.

## PRD 4 Components

- Unit and API contract tests for security, config, token, collector fallback, and CLI behavior.
- GitHub Actions PR/main workflow for formatting, tests, multi-platform build matrix, Docker build validation, and packaging script syntax checks.
- GitHub Actions release workflow for tagged releases, supported OS/architecture binaries, checksums, Docker image build, and GitHub release upload.
- Release metadata and architecture documentation.

## Error Handling

Security failures return JSON `401 Unauthorized` without leaking token values. Misconfiguration, such as `0.0.0.0` without authentication, prevents startup with a clear error. Token file permissions are enforced when files are created. Diagnostics report paths, bind state, schema version, collector status, and token-file presence without printing token contents.

## Testing

Production code changes are test-first. Script and workflow artifacts are verified with shell or PowerShell parser checks where available. Full verification runs `gofmt`, `go test ./...`, `go build`, shell syntax checks, and workflow YAML checks using a local parser when available.

## Later Hardening

This pass avoids heavyweight platform integrations that require external installers or admin access during tests. Later iterations can replace script templates with signed native installers, add OS-specific collectors, and publish multi-architecture Docker manifests from real CI credentials.
