# FleetPulse Disk Health Behavior

Disk health data is platform and permission dependent.

The v1 schema includes a `health` object per volume with status, optional temperature, and warning strings. On macOS, FleetPulse uses the built-in `diskutil info` output when available. Linux reports mounted volume capacity and marks disk health as unsupported until a safe SMART collector is added. Other platforms and unsupported device types keep explicit unsupported or unavailable health status.

Operators should expect:

- `unsupported` when FleetPulse has no collector for the current platform or device.
- `unavailable` when a collector exists but permissions or runtime state block access.
- `available` when health data is collected successfully.
