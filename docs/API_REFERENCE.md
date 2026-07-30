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

## Memory Fields

The `memory` section reports host memory byte counters and utilization:

- `total_bytes`: total memory in bytes.
- `used_bytes`: memory used in bytes.
- `free_bytes`: free memory in bytes.
- `available_bytes`: memory available to applications in bytes.
- `percent_used`: used memory as a percentage of total memory.

## Linux GPU Discovery

Linux GPU detection tries Raspberry Pi/VideoCore, NVIDIA, AMD, Intel, and generic DRM/sysfs paths. Discovery is cached in `/var/lib/fleetpulse/gpu-discovery.json`; remove that file to force rediscovery.

GPU metric coverage depends on the detected vendor, installed tools, drivers, and permissions. `unsupported` means no detector matched. `unavailable` means a detector exists but runtime data could not be read.
