# FleetPulse PRD 1 Core Agent and API Design

## Scope

This design implements the first product slice from `docs/01_FleetPulse_PRD_Core_Agent_and_API.md`: a lightweight Go agent exposing a stable, read-only JSON API for system telemetry. It does not implement PRD 2 authentication, token lifecycle, installers, Docker packaging, or CI release automation. Those later PRDs must preserve the schema introduced here.

## Approach

FleetPulse will be a single Go binary with a small HTTP server and a collector layer. The server owns routing and JSON response formatting. Collectors return normalized metric sections that explicitly distinguish `available`, `unsupported`, `unavailable`, `host`, and `container` scope states. Initial collectors favor standard library APIs so the project can build and test without external dependencies; platform-specific enhancements can be added behind the same interfaces later.

## API

The v1 API exposes these JSON endpoints:

- `GET /health`
- `GET /v1/stats`
- `GET /v1/cpu`
- `GET /v1/memory`
- `GET /v1/disks`
- `GET /v1/gpu`
- `GET /v1/system`

Every response includes `timestamp`, `schema_version`, and relevant metric data. `/v1/stats` returns a composed snapshot containing identity, platform, CPU, memory, disks, GPU, uptime, and load data.

## Data Model

Metric sections expose a consistent status envelope:

- `status`: `available`, `unsupported`, or `unavailable`
- `scope`: `host`, `container`, or `unavailable`
- `error`: optional operator-facing reason when unavailable

Known numeric values use pointers or nullable JSON fields so zero is never confused with unavailable. Unsupported collectors still return a valid section instead of failing the full response.

## Components

- `cmd/fleetpulse`: command entry point and HTTP server startup.
- `internal/api`: HTTP routing, handlers, and JSON helpers.
- `internal/collector`: collector interfaces and snapshot orchestration.
- `internal/schema`: public response structs and stable JSON contract.

## Error Handling

The API should keep serving partial telemetry if one collector fails. A collector failure marks only its section as `unavailable` and includes a short error string. Unknown routes return JSON 404 responses. Method mismatches return JSON 405 responses.

## Testing

Tests cover the API contract, route behavior, snapshot composition, and status semantics. Implementation follows red-green-refactor: write a failing test for each behavior, watch it fail, then add the smallest production code needed to pass.

## Future PRD Compatibility

The server will accept bind address and port flags now, but authentication remains out of scope until PRD 2. The collector boundaries are intentionally narrow so token handling, caching, timeouts, diagnostics, installers, Docker packaging, and CI can be layered on without changing the v1 response contract.
