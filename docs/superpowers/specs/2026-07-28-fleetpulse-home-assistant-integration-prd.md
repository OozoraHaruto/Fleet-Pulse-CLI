# FleetPulse Home Assistant HACS Integration PRD

## Summary

Build a HACS-installable Home Assistant custom integration for FleetPulse. The integration lets a user connect Home Assistant to a FleetPulse agent by entering an IP address, port, and bearer token, then creates Home Assistant sensor entities from the FleetPulse `GET /v1/stats` payload.

Although users may casually call this a plugin, the deliverable is a Home Assistant custom integration because it talks to an API and creates entities. It must install through HACS as an `integration` repository type, not as a dashboard/plugin repository type.

## Goals

- Install from HACS as a custom repository and work from `custom_components/fleetpulse`.
- Configure the FleetPulse endpoint from the Home Assistant UI using `ip`, `port`, and `token`.
- Validate the connection and token during setup before creating the config entry.
- Poll only `GET /v1/stats`, because that response contains the full telemetry snapshot.
- Represent every scalar value in the FleetPulse stats API as a Home Assistant sensor when it can be addressed safely.
- Assign correct Home Assistant sensor metadata: device class, state class, native unit, entity category, and diagnostics classification where appropriate.
- Let users edit sensor filtering after setup through an options flow.
- Support both exclusion and inclusion filtering:
  - `exclude`: show every discovered sensor except selected ignored sensors.
  - `include`: show only selected sensors.
- Preserve existing Home Assistant entity registry behavior as much as possible when options change.
- Provide CI with HACS validation, hassfest validation, Python linting, typing, tests, and packaging checks.

## Non-Goals

- Do not build a Lovelace dashboard card.
- Do not call FleetPulse endpoints other than `/v1/stats` for normal telemetry.
- Do not control, configure, restart, or mutate the FleetPulse agent.
- Do not support YAML-only configuration in the first release.
- Do not require a cloud account or external service.
- Do not create binary sensors, buttons, switches, or diagnostics panels in the first release; all FleetPulse API values are modeled as `sensor` entities.

## Users and Use Cases

Primary users run FleetPulse on a host, server, Raspberry Pi, VM, container, or workstation and want that telemetry inside Home Assistant for dashboards, automations, and history.

Core use cases:

- Add a FleetPulse host from the Home Assistant Integrations UI.
- Store the API token securely in the Home Assistant config entry.
- See CPU, memory, disk, GPU, system, target, schema, and status values as sensors.
- Hide noisy sensors such as per-core CPU values, individual mount points, status details, or GPU details.
- Switch later from "show all except ignored" to "only show selected" without deleting and re-adding the integration.

## Product Decisions

### Integration Type

The repository is a Home Assistant custom integration with this structure:

```text
custom_components/fleetpulse/
  __init__.py
  api.py
  config_flow.py
  const.py
  coordinator.py
  entity.py
  manifest.json
  sensor.py
  strings.json
  translations/en.json
hacs.json
README.md
.github/workflows/validate.yml
tests/
```

`manifest.json` sets `config_flow: true` and `integration_type: hub`, because one FleetPulse endpoint can expose multiple related entities for one host.

### Setup Flow

The setup form asks for:

- `ip`: host or IP address, required.
- `port`: integer TCP port, required, default `35338`.
- `token`: bearer token, required, stored in the config entry.
- `name`: optional display name, defaulting to the target hostname returned by FleetPulse.
- `ssl`: optional boolean, default `false`.

On submit, the integration calls:

```http
GET http://<ip>:<port>/v1/stats
Authorization: Bearer <token>
```

If `ssl` is true, use `https`. The setup flow validates that the response is JSON, has `schema_version`, `timestamp`, `target`, and known metric section objects, and is not an authentication failure.

Failure handling:

- Connection timeout, DNS failure, refused connection: show `cannot_connect`.
- HTTP `401`: show `invalid_auth`.
- Non-JSON or schema mismatch: show `invalid_response`.
- Any other unexpected exception: show `unknown`.

The unique ID should be stable per FleetPulse target. Prefer `target.machine_id` when present, otherwise use normalized `target.hostname`, `target.platform`, `target.architecture`, `ip`, and `port`.

### Reconfiguration

The integration must support a reconfigure flow for setup data that may change:

- `ip`
- `port`
- `token`
- `ssl`

The reconfigure flow validates the new settings against `/v1/stats`, updates the existing config entry, and reloads it. It must not create a second entry for the same target.

### Options Flow

The options flow controls entity filtering after the first successful fetch.

Options:

- `sensor_filter_mode`: enum, values `exclude` and `include`, default `exclude`.
- `selected_sensor_keys`: list of stable sensor keys.

Behavior:

- In `exclude` mode, all currently discoverable sensor keys are enabled except keys in `selected_sensor_keys`.
- In `include` mode, only keys in `selected_sensor_keys` are enabled.
- The options flow must be editable from Home Assistant after setup.
- The flow should show the latest discovered sensor keys grouped by category: Target, API, System, CPU, Memory, Disk, Disk Health, GPU, Status.
- When a sensor disappears because the API no longer returns the backing object, the entity should become unavailable rather than being deleted automatically.
- When a new sensor appears and the mode is `exclude`, it should be added automatically unless explicitly ignored.
- When a new sensor appears and the mode is `include`, it should remain hidden until selected.

## API Contract

The integration consumes FleetPulse schema version `v1` from:

```http
GET /v1/stats
```

Required top-level fields:

- `timestamp`
- `schema_version`
- `target`
- `system`
- `cpu`
- `memory`
- `disks`
- `gpu`

Each metric section includes:

- `status`: `available`, `unsupported`, or `unavailable`
- `scope`: `host`, `container`, or `unavailable`
- `error`: optional string

The integration must tolerate unknown extra fields by ignoring them. Missing known nullable values must result in unavailable sensors, not setup failure, as long as the top-level response is otherwise valid.

## Entity and Device Model

Each FleetPulse config entry creates one Home Assistant device representing the monitored host.

Device identifiers:

- Domain: `fleetpulse`
- Identifier: `target.machine_id` if present.
- Fallback identifier: normalized `<hostname>-<platform>-<architecture>-<ip>-<port>`.

Device metadata:

- Name: user-provided name or `target.hostname`.
- Manufacturer: `FleetPulse`.
- Model: `<platform> <architecture>`.
- Software version: `schema_version`.
- Configuration URL: `http://<ip>:<port>/v1/stats` or `https://<ip>:<port>/v1/stats`.

Entity unique IDs use:

```text
<device_identifier>_<sensor_key>
```

Sensor keys are lowercase, ASCII, and stable across restarts. Dynamic list keys must include a deterministic index plus a sanitized identity when available.

Examples:

- `cpu_utilization_percent`
- `memory_total_bytes`
- `disk_0_root_percent_used`
- `disk_0_root_health_temperature_celsius`
- `gpu_0_nvidia_rtx_4000_temperature_celsius`

## Sensor Availability Rules

- A sensor is available when its backing value exists and its parent section is not `unavailable`.
- Numeric JSON `0` is a valid value and must not be treated as missing.
- JSON `null`, absent fields, and unavailable parent sections make the affected sensor unavailable.
- `status`, `scope`, and `error` sensors remain available when the parent section exists, because they explain availability.
- `unsupported` sections should still expose their status and scope sensors.
- String sensors return strings exactly as reported, except empty strings make optional metadata sensors unavailable.

## Sensor Mapping

### API and Target Sensors

| Sensor key | API field | Device class | State class | Unit | Entity category |
| --- | --- | --- | --- | --- | --- |
| `api_timestamp` | `timestamp` | `timestamp` | none | none | diagnostic |
| `api_schema_version` | `schema_version` | none | none | none | diagnostic |
| `target_hostname` | `target.hostname` | none | none | none | diagnostic |
| `target_platform` | `target.platform` | enum | none | none | diagnostic |
| `target_architecture` | `target.architecture` | enum | none | none | diagnostic |
| `target_machine_id` | `target.machine_id` | none | none | none | diagnostic |

### System Sensors

| Sensor key | API field | Device class | State class | Unit | Entity category |
| --- | --- | --- | --- | --- | --- |
| `system_status` | `system.status` | enum | none | none | diagnostic |
| `system_scope` | `system.scope` | enum | none | none | diagnostic |
| `system_error` | `system.error` | none | none | none | diagnostic |
| `system_uptime_seconds` | `system.uptime_seconds` | duration | measurement | seconds | diagnostic |
| `system_load_1m` | `system.load_average.one_minute` | none | measurement | none | none |
| `system_load_5m` | `system.load_average.five_minutes` | none | measurement | none | none |
| `system_load_15m` | `system.load_average.fifteen_minutes` | none | measurement | none | none |

### CPU Sensors

| Sensor key | API field | Device class | State class | Unit | Entity category |
| --- | --- | --- | --- | --- | --- |
| `cpu_status` | `cpu.status` | enum | none | none | diagnostic |
| `cpu_scope` | `cpu.scope` | enum | none | none | diagnostic |
| `cpu_error` | `cpu.error` | none | none | none | diagnostic |
| `cpu_model` | `cpu.model` | none | none | none | diagnostic |
| `cpu_core_count` | `cpu.core_count` | none | measurement | cores | diagnostic |
| `cpu_utilization_percent` | `cpu.utilization_percent` | none | measurement | `%` | none |
| `cpu_core_<index>_utilization_percent` | `cpu.per_core_utilization_percent[index]` | none | measurement | `%` | none |

### Memory Sensors

| Sensor key | API field | Device class | State class | Unit | Entity category |
| --- | --- | --- | --- | --- | --- |
| `memory_status` | `memory.status` | enum | none | none | diagnostic |
| `memory_scope` | `memory.scope` | enum | none | none | diagnostic |
| `memory_error` | `memory.error` | none | none | none | diagnostic |
| `memory_total_bytes` | `memory.total_bytes` | data_size | measurement | bytes | none |
| `memory_used_bytes` | `memory.used_bytes` | data_size | measurement | bytes | none |
| `memory_free_bytes` | `memory.free_bytes` | data_size | measurement | bytes | none |
| `memory_available_bytes` | `memory.available_bytes` | data_size | measurement | bytes | none |

### Disk Sensors

Create one sensor set per `disks.volumes[index]`.

| Sensor key pattern | API field | Device class | State class | Unit | Entity category |
| --- | --- | --- | --- | --- | --- |
| `disks_status` | `disks.status` | enum | none | none | diagnostic |
| `disks_scope` | `disks.scope` | enum | none | none | diagnostic |
| `disks_error` | `disks.error` | none | none | none | diagnostic |
| `disk_<index>_<mount>_mount_point` | `disks.volumes[index].mount_point` | none | none | none | diagnostic |
| `disk_<index>_<mount>_filesystem_type` | `disks.volumes[index].filesystem_type` | enum | none | none | diagnostic |
| `disk_<index>_<mount>_total_bytes` | `disks.volumes[index].total_bytes` | data_size | measurement | bytes | none |
| `disk_<index>_<mount>_used_bytes` | `disks.volumes[index].used_bytes` | data_size | measurement | bytes | none |
| `disk_<index>_<mount>_free_bytes` | `disks.volumes[index].free_bytes` | data_size | measurement | bytes | none |
| `disk_<index>_<mount>_percent_used` | `disks.volumes[index].percent_used` | none | measurement | `%` | none |

### Disk Health Sensors

Create these sensors when `disks.volumes[index].health` exists. If `health` is absent, the health-specific sensors for that volume are unavailable.

| Sensor key pattern | API field | Device class | State class | Unit | Entity category |
| --- | --- | --- | --- | --- | --- |
| `disk_<index>_<mount>_health_status` | `disks.volumes[index].health.status` | enum | none | none | diagnostic |
| `disk_<index>_<mount>_health_temperature_celsius` | `disks.volumes[index].health.temperature_celsius` | temperature | measurement | celsius | none |
| `disk_<index>_<mount>_health_warnings` | `disks.volumes[index].health.warnings` joined with `; ` | none | none | none | diagnostic |

### GPU Sensors

Create one sensor set per `gpu.devices[index]`.

| Sensor key pattern | API field | Device class | State class | Unit | Entity category |
| --- | --- | --- | --- | --- | --- |
| `gpu_status` | `gpu.status` | enum | none | none | diagnostic |
| `gpu_scope` | `gpu.scope` | enum | none | none | diagnostic |
| `gpu_error` | `gpu.error` | none | none | none | diagnostic |
| `gpu_<index>_<identity>_vendor` | `gpu.devices[index].vendor` | enum | none | none | diagnostic |
| `gpu_<index>_<identity>_model` | `gpu.devices[index].model` | none | none | none | diagnostic |
| `gpu_<index>_<identity>_memory_total_bytes` | `gpu.devices[index].memory_total_bytes` | data_size | measurement | bytes | none |
| `gpu_<index>_<identity>_memory_used_bytes` | `gpu.devices[index].memory_used_bytes` | data_size | measurement | bytes | none |
| `gpu_<index>_<identity>_utilization_percent` | `gpu.devices[index].utilization_percent` | none | measurement | `%` | none |
| `gpu_<index>_<identity>_temperature_celsius` | `gpu.devices[index].temperature_celsius` | temperature | measurement | celsius | none |

## Polling and Coordinator

Use a single `DataUpdateCoordinator` per config entry.

Requirements:

- Poll interval default: 30 seconds.
- Timeout default: 10 seconds.
- Coordinator update method calls only `/v1/stats`.
- Set coordinator `always_update=False` if the parsed snapshot object supports equality comparison.
- Entity properties read from coordinator memory only and never perform I/O.
- Use Home Assistant's native aiohttp session helpers for async HTTP.
- Redact the token from logs, diagnostics, exceptions, and repair messages.

On HTTP errors:

- `401`: mark setup as reauthentication-required or start reauth flow after setup.
- `404`: mark coordinator update failed with a clear API path error.
- `5xx`: mark update failed and keep previous entity states until Home Assistant marks them unavailable.

## Validation and CI Requirements

Create `.github/workflows/validate.yml` with these jobs:

- HACS validation:
  - Trigger on `push`, `pull_request`, scheduled daily run, and `workflow_dispatch`.
  - Use `hacs/action`.
  - Set `category: integration`.
- Hassfest validation:
  - Use the Home Assistant hassfest action for custom integration validation.
- Python quality:
  - Install project test dependencies.
  - Run `ruff format --check`.
  - Run `ruff check`.
  - Run `mypy custom_components/fleetpulse tests`.
  - Run `pytest`.
- Package structure:
  - Verify `hacs.json` exists at repo root.
  - Verify `custom_components/fleetpulse/manifest.json` exists.
  - Verify `custom_components/fleetpulse/translations/en.json` exists.
  - Verify no token, fixture secret, or local Home Assistant config files are committed.

The repo should also include Dependabot updates for GitHub Actions and Python dependencies.

## HACS Repository Requirements

The repository must be public on GitHub for normal HACS use. It must include:

- GitHub repository description.
- GitHub repository topics, including `home-assistant`, `hacs`, `fleetpulse`, `sensor`, and `monitoring`.
- `README.md` with installation and setup instructions.
- Root `hacs.json` with at least `name`.
- GitHub releases for stable version discovery.
- Passing HACS and hassfest GitHub Actions.

Recommended root `hacs.json`:

```json
{
  "name": "FleetPulse",
  "homeassistant": "2025.1.0"
}
```

The minimum Home Assistant version can be adjusted during implementation if a required API needs a newer release.

## Error Handling and UX Copy

Setup errors:

- `cannot_connect`: "Could not connect to FleetPulse. Check the IP address, port, and protocol."
- `invalid_auth`: "FleetPulse rejected the token."
- `invalid_response`: "FleetPulse responded, but the response did not match the expected stats schema."
- `unknown`: "Unexpected error while connecting to FleetPulse."

Options errors:

- If no sensor keys are available yet, show a form that explains the integration needs one successful stats fetch before sensor selection can be edited.
- If saved sensor keys are no longer returned by the API, keep them in the options data and show them as stale selections where Home Assistant's form controls allow it.

## Testing Requirements

Unit tests must cover:

- API client builds the correct URL for HTTP and HTTPS.
- API client sends `Authorization: Bearer <token>`.
- API client maps timeout, auth, invalid JSON, and schema failures to typed exceptions.
- Config flow creates an entry after a valid stats response.
- Config flow rejects invalid auth and connection errors.
- Reconfigure updates the existing entry instead of creating a second entry.
- Options flow stores `exclude` and `include` mode correctly.
- Sensor description registry includes every field listed in this PRD.
- Numeric zero values remain valid sensor states.
- Missing nullable values produce unavailable sensors.
- Per-core, per-disk, disk-health, and per-GPU sensors use stable keys.
- Token values are never logged.

Integration-style tests must cover:

- A mocked `/v1/stats` response creates the expected entities.
- Changing options from `exclude` to `include` removes or disables unselected entities from the integration's active entity list without corrupting the config entry.
- Unsupported GPU still exposes `gpu_status` and `gpu_scope` sensors.

## Documentation Requirements

README must include:

- HACS custom repository installation steps.
- Home Assistant setup steps.
- Required FleetPulse URL and token configuration.
- Explanation of include and exclude sensor filtering.
- Example sensors users should expect to see.
- Troubleshooting for connection errors, invalid token, missing sensors, and unsupported collectors.
- A note that FleetPulse values are read-only sensors.

## Acceptance Criteria

- A user can add the repo to HACS as an integration and install it.
- Home Assistant discovers the custom integration after restart.
- The UI config flow accepts valid `ip`, `port`, and `token` values.
- A valid FleetPulse `/v1/stats` response creates sensor entities for all mapped API fields.
- The options flow can hide selected sensors in `exclude` mode.
- The options flow can show only selected sensors in `include` mode.
- Reconfiguration can update endpoint details and token without deleting the config entry.
- HACS validation passes.
- Hassfest validation passes.
- Linting, typing, and tests pass in GitHub Actions.

## References

- HACS publishing requirements: https://hacs.xyz/docs/publish/start/
- HACS GitHub Action: https://hacs.xyz/docs/publish/action/
- Home Assistant config flow: https://developers.home-assistant.io/docs/core/integration/config_flow/
- Home Assistant options flow: https://developers.home-assistant.io/docs/core/integration/options_flow/
- Home Assistant fetching data and `DataUpdateCoordinator`: https://developers.home-assistant.io/docs/integration_fetching_data/
- Home Assistant sensor entity conventions: https://developers.home-assistant.io/docs/core/entity/sensor/
- Home Assistant integration manifest: https://developers.home-assistant.io/docs/creating_integration_manifest/
- Home Assistant integration file structure: https://developers.home-assistant.io/docs/creating_integration_file_structure/
