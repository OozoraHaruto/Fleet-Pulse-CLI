# FleetPulse Configuration Reference

FleetPulse starts with safe defaults, then applies an optional JSON config file, environment variables, and command-line flags.

## JSON Fields

- `addr`: bind address, default `0.0.0.0:35338`.
- `auth_enabled`: whether bearer authentication is required, default `true`.
- `token_file`: path to the protected bearer token file, defaulting under the current user's config directory.
- `cache_ttl`: snapshot cache TTL, such as `10s`.
- `collector_timeout`: per-collector timeout, such as `2s`.
- `enabled_collectors`: collector names to enable.
- `log_level`: `debug`, `info`, `warn`, or `error`.
- `schema_version`: defaults to `v1`.
- `service_name`: service name used in diagnostics and installers.
- `deployment_target`: optional `linux`, `windows`, `macos`, or `docker` override.
- `disk_health_enabled`: enables disk health collection when platform support exists.

## Environment Variables

- `FLEETPULSE_ADDR`
- `FLEETPULSE_AUTH_ENABLED`
- `FLEETPULSE_TOKEN_FILE`
- `FLEETPULSE_CACHE_TTL`
- `FLEETPULSE_COLLECTOR_TIMEOUT`

## Safety Rule

Binding to `0.0.0.0`, `::`, or a non-loopback interface requires `auth_enabled: true`.
