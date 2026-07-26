# Linux GPU Discovery Cache Design

## Goal

FleetPulse should collect Linux GPU identity and best-effort metrics across common GPU families without probing every request. The collector should discover the working GPU path once, persist that discovery in FleetPulse state, reuse it while valid, and rediscover when the cached path stops working or becomes stale.

This design covers Linux only. macOS keeps the existing Darwin GPU collector, and other platforms continue to report explicit unsupported GPU status.

## Current Problem

Linux currently reports:

```json
{
  "status": "unsupported",
  "scope": "unavailable",
  "error": "gpu collection requires a vendor-specific collector"
}
```

That is too conservative for machines where FleetPulse can identify GPUs through standard Linux surfaces or installed vendor tools. However, GPU collection is not vendor-neutral: Raspberry Pi, NVIDIA, AMD, and Intel expose different files, commands, permissions, and metric coverage. Re-running all probes on every `/v1/gpu` or `/v1/stats` request would be slow and noisy, especially on small ARM Linux devices.

## Non-Goals

- Do not require privileged access.
- Do not install vendor packages.
- Do not make GPU collection failure fail the full snapshot.
- Do not change the v1 JSON schema.
- Do not remove explicit `unsupported` and `unavailable` semantics.

## Architecture

Add a Linux GPU detector registry under `internal/collector`.

Each detector has:

- `Name`: stable identifier persisted in the cache.
- `Discover(ctx)`: checks whether this detector applies and returns cached metadata needed for future collection.
- `Collect(ctx, discovery)`: uses a previous discovery result to return a `schema.GPUSection`.

Detector order:

1. Raspberry Pi / VideoCore
2. NVIDIA
3. AMD
4. Intel
5. generic DRM/sysfs fallback

The order favors specific detectors before broad DRM/sysfs fallback so richer vendor data wins when available.

## Persistent Cache

Linux writes discovery state to:

```text
/var/lib/fleetpulse/gpu-discovery.json
```

For this implementation, use the existing Linux service state location and degrade gracefully if it is not writable. A future first-class state directory config can move this path without changing the cache format.

Cache fields:

- `version`: discovery cache format version.
- `detector`: detector name, such as `raspberry-pi`, `nvidia`, `amd`, `intel`, `drm`.
- `status`: `available` or `unsupported`.
- `vendor`, `model`: stable identity strings when available.
- `devices`: detector-specific command names and sysfs paths needed by collection.
- `host_fingerprint`: hostname, platform, architecture, and relevant device IDs or sysfs paths.
- `discovered_at`: timestamp.
- `last_failure_at`: optional timestamp.
- `failure_count`: consecutive cached-collector failures.

The cache file must not contain secrets. Write it with owner-readable permissions only where possible.

## Runtime Flow

On `collectGPU(ctx)` for Linux:

1. Load cached discovery if present.
2. If the cache is fresh and host/device fingerprint still matches, use its detector immediately.
3. If collection succeeds, return the collected `schema.GPUSection` and clear failure state.
4. If collection fails, return `unavailable` for this request, increment failure state, and rediscover once the failure threshold is reached.
5. If cache is missing, stale, incompatible, or fingerprint-mismatched, run discovery.
6. If discovery finds a detector, persist it and collect with it.
7. If discovery finds no supported detector, persist an `unsupported` result with a shorter TTL and return `unsupported`.

Suggested TTLs:

- Available discovery: 24 hours.
- Unsupported discovery: 1 hour.
- Rediscovery threshold: 2 consecutive cached collection failures.

## Detector Behavior

### Raspberry Pi / VideoCore

Discovery looks for Raspberry Pi identity in `/proc/device-tree/model`, `/sys/firmware/devicetree/base/model`, or compatible VideoCore DRM device names. Collection returns vendor/model when identified. Metrics are best effort because utilization and memory information vary by Pi generation and distro.

### NVIDIA

Discovery prefers `nvidia-smi` when present. Collection parses query output for model, memory total, memory used, utilization, and temperature. If `nvidia-smi` is present but fails because of permissions or driver state, return `unavailable`.

### AMD

Discovery uses DRM/sysfs vendor IDs for AMD devices. Collection returns vendor/model from sysfs where available and best-effort memory/utilization/temperature only when readable without elevated privileges.

### Intel

Discovery uses DRM/sysfs vendor IDs for Intel devices. Collection returns vendor/model from sysfs where available and best-effort utilization/temperature only when readable without elevated privileges.

### Generic DRM/sysfs

Fallback discovery identifies GPU-like DRM devices and returns vendor/model when sysfs exposes them. Metrics can remain null. This still improves API usefulness by reporting `available` GPU identity rather than `unsupported`.

## Error Handling

- Missing cache file is normal.
- Invalid cache JSON is ignored and replaced on successful discovery.
- Cache write failures do not fail collection; they leave the request result based on live discovery.
- Detector command failures during cached collection return `unavailable`, not `unsupported`, because a collector exists but runtime state blocked it.
- No matching detector returns `unsupported`.
- Context cancellation or timeout should stop command-based probes.

## Testing

Add tests for:

- Discovery cache load/save and invalid JSON handling.
- Fresh available cache reuses the saved detector without running all probes.
- Stale or fingerprint-mismatched cache triggers rediscovery.
- Cached collection failure increments failure count and switches to rediscovery at the threshold.
- Unsupported discovery is cached and expires sooner than available discovery.
- Linux detector parsers for Raspberry Pi model files, NVIDIA `nvidia-smi` query output, AMD/Intel sysfs vendor IDs, and generic DRM fallback identity.
- Linux `collectGPU` returns `available`, `unsupported`, or `unavailable` without panicking when cache or system files are missing.

## Documentation

Update README, API, configuration or troubleshooting docs to explain:

- Linux GPU detection is cached.
- GPU metrics vary by vendor and permissions.
- The cache can be deleted to force rediscovery.
- Unsupported means no detector matched; unavailable means the detector exists but could not read runtime data.
