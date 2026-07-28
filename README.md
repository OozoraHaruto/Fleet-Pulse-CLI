# FleetPulse

FleetPulse is a lightweight cross-platform system telemetry agent. It exposes host or container stats through a stable JSON HTTP API and is designed for polling by fleet monitoring systems.

The project currently includes the core agent, API contract, authentication/configuration support, token lifecycle commands, packaging artifacts, Docker deployment examples, and GitHub Actions CI/release workflows.

## Features

- Read-only HTTP API for health and telemetry.
- Stable v1 JSON schema with explicit availability states.
- CPU, memory, disk, disk health, GPU, uptime, and load sections.
- Host/container scope labeling for metrics.
- Bearer-token authentication for network-exposed deployments.
- Protected token file provisioning, show, and rotation.
- Configurable bind address, cache TTL, collector timeout, collectors, log level, and deployment target.
- Linux systemd, macOS launchd, Windows service, and Docker packaging artifacts.
- CI and release automation for supported OS/architecture builds.

## Current Collector Status

FleetPulse already returns a stable API shape. On macOS, the collector reports host identity, platform, architecture, CPU core count, memory totals, mounted volume capacity, GPU identity, uptime, and load averages. On Linux, including arm64 Raspberry Pi deployments, it reports host identity, platform, architecture, CPU core count, memory totals, mounted volume capacity, uptime, and load averages. Other platforms keep explicit unsupported/unavailable sections for metrics that still need platform-specific collectors.

That means clients can integrate against the v1 schema now without treating missing hardware or unsupported metrics as API failures.

## Quick Start

Build the agent:

```sh
go build -o fleetpulse ./cmd/fleetpulse
```

Run with the default network bind and bearer authentication:

```sh
./fleetpulse serve
```

Check health:

```sh
curl http://127.0.0.1:35338/health
```

Fetch the full telemetry snapshot:

```sh
curl -H "Authorization: Bearer <token>" http://127.0.0.1:35338/v1/stats
```

## Authentication

FleetPulse binds to `0.0.0.0:35338` with authentication enabled by default.
It refuses to bind to a non-local interface without authentication enabled.

Override the default bind explicitly:

```sh
./fleetpulse serve \
  -addr 0.0.0.0:35338 \
  -auth=true \
  -token-file /var/lib/fleetpulse/token
```

On first authenticated startup, FleetPulse provisions a protected token file and prints the initial bearer token once.

Use authenticated requests:

```sh
curl -H "Authorization: Bearer <token>" http://127.0.0.1:35338/v1/stats
```

Token operations:

```sh
fleetpulse token show -token-file /var/lib/fleetpulse/token
fleetpulse token rotate -token-file /var/lib/fleetpulse/token
```

## API

All v1 responses are JSON.

- `GET /health`
- `GET /v1/stats`
- `GET /v1/cpu`
- `GET /v1/memory`
- `GET /v1/disks`
- `GET /v1/gpu`
- `GET /v1/system`

Metric sections include:

- `status`: `available`, `unsupported`, or `unavailable`
- `scope`: `host`, `container`, or `unavailable`
- `error`: optional reason for unavailable data

On Linux, GPU discovery is cached at `/var/lib/fleetpulse/gpu-discovery.json` so FleetPulse can reuse the detected Raspberry Pi, NVIDIA, AMD, Intel, or DRM/sysfs path instead of probing every request. Delete that file to force GPU rediscovery.

See [docs/API_REFERENCE.md](docs/API_REFERENCE.md) for details.

## Configuration

Show the effective config:

```sh
fleetpulse config show
```

Use a JSON config file:

```sh
fleetpulse serve -config /etc/fleetpulse/fleetpulse.json
```

Supported environment overrides:

- `FLEETPULSE_ADDR`
- `FLEETPULSE_AUTH_ENABLED`
- `FLEETPULSE_TOKEN_FILE`
- `FLEETPULSE_CACHE_TTL`
- `FLEETPULSE_COLLECTOR_TIMEOUT`

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md).

## Diagnostics

Run:

```sh
fleetpulse diagnose -config /etc/fleetpulse/fleetpulse.json
```

Diagnostics report config paths, bind safety, collector settings, schema version, release metadata, and token-file presence. Token values are not printed.

## Installation

Install the latest Linux or macOS release from GitHub:

```sh
curl -fsSL https://raw.githubusercontent.com/OozoraHaruto/Fleet-Pulse-CLI/main/install.sh | sh
```

To pin a version:

```sh
curl -fsSL https://raw.githubusercontent.com/OozoraHaruto/Fleet-Pulse-CLI/main/install.sh | FLEETPULSE_VERSION=v1.2.3 sh
```

To test the newest prerelease before a stable release exists:

```sh
curl -fsSL https://raw.githubusercontent.com/OozoraHaruto/Fleet-Pulse-CLI/main/install.sh | FLEETPULSE_ALLOW_PRERELEASE=true sh
```

FleetPulse includes lifecycle artifacts for each supported target:

- Linux: [packaging/linux](packaging/linux)
- macOS: [packaging/macos](packaging/macos)
- Windows: [packaging/windows](packaging/windows)
- Docker: [Dockerfile](Dockerfile) and [deploy/docker-compose.yml](deploy/docker-compose.yml)

Detailed guides:

- [docs/INSTALLATION.md](docs/INSTALLATION.md)
- [docs/UNINSTALLATION.md](docs/UNINSTALLATION.md)
- [docs/DOCKER.md](docs/DOCKER.md)
- [docs/UPGRADE_ROLLBACK.md](docs/UPGRADE_ROLLBACK.md)

## Testing

Run the full test suite:

```sh
go test ./...
```

Build the binary:

```sh
go build -o fleetpulse ./cmd/fleetpulse
```

The CI workflow runs formatting checks, unit/API tests, cross-platform builds, packaging script validation, and Docker build validation.

## Releases

Releases are intentionally infrequent:

- `version.txt` is the stable release source of truth and must contain a value such as `v1.0.0`.
- Pull requests run CI and build prerelease-named artifacts without publishing a GitHub release.
- Pushes to `main` run the release workflow as a GitHub prerelease using the main-branch version plus a unique `-ci.<run>.<sha>` suffix.
- Pushing a semver tag such as `v1.2.3`, or starting the release workflow manually, publishes the final version from `version.txt`.

Release artifacts include Linux, macOS, and Windows archives, SHA-256 checksum files, platform-specific installer files, operator docs, and a versioned GHCR Docker image.

See [docs/RELEASE.md](docs/RELEASE.md).

## Documentation

- [CLI reference](docs/CLI_REFERENCE.md)
- [API reference](docs/API_REFERENCE.md)
- [Configuration reference](docs/CONFIGURATION.md)
- [Token operations](docs/TOKEN_OPERATIONS.md)
- [Permissions guide](docs/PERMISSIONS.md)
- [Disk health behavior](docs/DISK_HEALTH.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Release guide](docs/RELEASE.md)

The original product requirements are in [docs](docs).
