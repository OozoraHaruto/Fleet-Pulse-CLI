# FleetPulse PRD 1

## Core Agent and API

# 1. Product overview
FleetPulse is a lightweight, cross-platform system stats agent that exposes machine and container telemetry through an HTTP API. It must support four first-class deployment targets: Linux host, Windows host, macOS host, and Docker container.

# 2. Problem statement
Fleet operators need a consistent way to collect system statistics from heterogeneous environments without building separate collectors for each operating system or deployment model.

# 3. Goals
- Provide one agent that works across Linux, Windows, macOS, and Docker.
- Expose host or container stats through an HTTP API.
- Return normalized, machine-readable JSON.
- Support CPU, memory, disks, disk health, GPU, uptime, and load where available.
- Keep the agent lightweight and suitable for polling by fleet monitoring systems.

# 4. Non-goals
- No cloud backend or hosted dashboard in v1.
- No alerting engine, RBAC, or user management in v1.
- No attempt to perfectly normalize every GPU metric across vendors and operating systems.
- No TLS requirements in this PRD set.

# 5. Functional requirements

## 5.1 Data collection
- CPU model, core count, utilization, and per-core utilization where available.
- Memory total, used, free, and available.
- Mounted volumes and filesystem usage including mount point, filesystem type, total size, used size, free size, and percent used.
- Disk health where available, including health status, temperature, SMART-style indicators, and warning flags.
- Uptime and load average where supported.
- GPU vendor, model, memory totals and usage, utilization, and temperature where available.

## 5.2 Deployment targets
- Linux host.
- Windows host.
- macOS host.
- Docker container.
Each target may expose a different subset of metrics, but the API contract must remain stable and clearly indicate metric availability and scope.

## 5.3 API
- GET /health
- GET /v1/stats
- GET /v1/cpu
- GET /v1/memory
- GET /v1/disks
- GET /v1/gpu
- GET /v1/system
The exact endpoint layout may be flattened or nested, but it must remain stable once released.

## 5.4 Output format
- JSON only for v1.
- Responses include a timestamp.
- Responses include target identity and platform information.
- Responses distinguish unsupported, unavailable, zero, host-scope, and container-scope values.

## 5.5 Scope awareness
FleetPulse must explicitly label whether returned metrics are host-level, container-level, or unavailable. This is especially important for Docker, where some values may reflect the container’s view rather than the underlying host.

# 6. Acceptance criteria
- A single JSON response can return CPU, memory, disk, disk health, GPU, uptime, and load data where available.
- The same API schema works across Linux, Windows, macOS, and Docker.
- Unsupported metrics are represented explicitly instead of failing the whole response.

# 7. Milestone
PRD 1 is complete when the core collector, normalized schema, and read-only API contract are implemented and documented.
