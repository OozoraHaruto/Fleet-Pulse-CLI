# FleetPulse Disk Health Behavior

Disk health data is platform and permission dependent.

The v1 schema includes a `health` object per volume with status, optional temperature, and warning strings. In the current implementation, disk health is represented as unsupported until platform-specific collectors are added.

Operators should expect:

- `unsupported` when FleetPulse has no collector for the current platform or device.
- `unavailable` when a collector exists but permissions or runtime state block access.
- `available` when health data is collected successfully.
