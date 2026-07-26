# FleetPulse PRD 2

## Security and Operations

# 1. Product scope
This PRD defines how FleetPulse behaves when network-exposed, how tokens are provisioned and stored, how targets are identified, and how the agent behaves under partial failure or limited permissions.

# 2. Security requirements

## 2.1 Network exposure
- FleetPulse must support binding to 0.0.0.0 or another configured non-local interface.
- If bound to any non-local interface, authentication must be enabled.
- The API must not be exposed unauthenticated on a network interface.

## 2.2 Authentication
- The initial authentication mechanism is a static bearer token.
- Clients send Authorization: Bearer <token>.
- Missing or invalid tokens return 401 Unauthorized.

## 2.3 Secrets handling
- Tokens must not be embedded in the binary.
- Tokens must not be hardcoded in the image.
- Tokens must not be repeatedly emitted in runtime logs.

# 3. Token provisioning requirements
- On first install or first startup, FleetPulse generates a random token.
- The token is displayed once during installation or first startup.
- The token is written to a protected local secret file.
- The token is retrievable later only through an admin-level CLI command.
- Native hosts store the token in a protected OS-specific config or secret file.
- Docker stores the token outside the container filesystem in a persistent mounted volume or equivalent external secret mechanism.
- Container replacement or image upgrades must not invalidate the token unless the operator explicitly rotates it.

# 4. Privilege and permission model
- Native installations should run with the lowest practical privileges.
- Collectors that require elevated access must be isolated and optional where possible.
- Docker deployments should document any extra capabilities, mounts, or privileges needed for host-level visibility.
- Permission failures must be reported as unavailable or partially unavailable, not as total agent failure.

# 5. Collection behavior
- Support a configurable polling interval or cache TTL.
- Prevent a slow or hung collector from blocking all responses indefinitely.
- Use timeouts for individual collectors.
- Cache recent results when appropriate to reduce overhead.
- Allow the API to return the most recent successful snapshot if a live collection fails.

# 6. Identity and inventory requirements
- Include hostname or container name where available.
- Include OS/platform and version information.
- Include a stable machine or container identifier if available.
- Provide an identifier suitable for fleet inventory use.
- Make clear when a reset, reinstall, or container replacement changes identity.

# 7. Logging and diagnostics requirements
- Structured logs should be available.
- Logs should include collector failures, startup state, and service lifecycle events.
- Sensitive values such as tokens must not be repeated in logs.
- A diagnostics command or report should be available in the CLI.
- Health checks should indicate whether the agent is running and whether collection is succeeding.

# 8. Upgrade and rollback requirements
- Native upgrades should preserve config and token state by default.
- Docker upgrades should preserve config and token state through mounted volumes or external secret handling.
- Upgrades must not require re-provisioning unless explicitly requested.
- Rollback should be possible by reinstalling the prior version with the same preserved state.
- Upgrade steps should be deterministic and documented.

# 9. Configuration
- Bind address and port.
- Authentication enabled/disabled.
- Token file path.
- Polling interval or cache TTL.
- Enabled collectors.
- Log level.
- Schema version.
- Service name.
- Deployment target override if needed for diagnostics.
- Permission hints or target-specific collector flags.
- Disk health collection enablement.
- Collector timeout values.

# 10. Operational assumptions
- The agent is read-only.
- The agent does not require outbound network access for normal operation.
- The agent should not depend on custom kernel modules.
- Secrets should not be emitted repeatedly.
- Default behavior should be safe and predictable.
- Supported architectures should be documented explicitly.
- The product should degrade gracefully when a metric is unavailable.

# 11. Acceptance criteria
- A network-exposed instance rejects requests without a valid bearer token.
- A token survives normal upgrades and Docker image replacement when the volume persists.
- Collector failures do not cause total API failure.
- Diagnostics and logs help operators identify failures quickly.
