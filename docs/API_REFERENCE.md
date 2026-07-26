# FleetPulse API Reference

All v1 responses are JSON.

## Endpoints

- `GET /health`
- `GET /v1/stats`
- `GET /v1/cpu`
- `GET /v1/memory`
- `GET /v1/disks`
- `GET /v1/gpu`
- `GET /v1/system`

When authentication is enabled, send:

```http
Authorization: Bearer <token>
```

Missing or invalid credentials return `401 Unauthorized` with a JSON error body.

## Availability Semantics

Each metric section includes:

- `status`: `available`, `unsupported`, or `unavailable`.
- `scope`: `host`, `container`, or `unavailable`.
- `error`: optional reason for unavailable data.

Zero values remain distinct from unavailable values because nullable numeric fields are encoded as JSON `null` when not known.
