# FleetPulse PRD 4

## Testing, CI/CD, and Release

# 1. Product scope
This PRD defines automated verification, GitHub Actions pipelines, release gating, and operational quality checks required to ship FleetPulse safely.

# 2. Testing requirements

## 2.1 Unit tests
- Collectors.
- Schema normalization.
- Token generation and rotation.
- Config parsing.
- CLI commands.
- API response formatting.
- Error handling and fallback behavior.

## 2.2 Integration tests
- Linux host collection.
- Windows host collection.
- macOS host collection.
- Docker container collection.
- Disk usage reporting.
- Disk health reporting where supported.
- GPU collection where supported.
- Service startup and shutdown.
- Install and uninstall flows.
- Upgrade preserving config and token state.

## 2.3 API contract tests
- Response schema stability.
- Required fields are present.
- Versioned endpoints behave consistently.
- Unsupported metrics are represented correctly.
- Partial collector failure does not break the full payload.

## 2.4 CLI tests
- Install.
- Uninstall.
- Upgrade.
- Status.
- Token show.
- Token rotate.
- Diagnose.
- Config get/set.

## 2.5 Platform matrix tests
- Linux.
- Windows.
- macOS.
- Docker.
- x64 and arm64 where supported.
- Systems with and without GPU.
- Systems with one or multiple disks.
- Restricted-permission environments.

## 2.6 Reliability tests
- Collector timeout handling.
- Partial failure recovery.
- Repeated polling.
- Service restart behavior.
- Container restart behavior.
- Uninstall cleanup.
- Upgrade rollback behavior.

## 2.7 Packaging validation
- Binaries start successfully after installation.
- Services register correctly.
- Uninstall removes services correctly.
- Docker images run with documented volumes and configuration.
- Token persistence works across upgrades.

# 3. GitHub Actions CI/CD requirements

## 3.1 Pull request pipeline
- Formatting checks.
- Linting.
- Unit tests.
- Schema or contract tests where available.
- Platform-relevant build checks.

## 3.2 Main branch pipeline
- Full automated test suite.
- Integration tests where feasible.
- Packaging validation.
- Docker build validation.
- Artifact generation for preview or release candidates if configured.

## 3.3 Release pipeline
- Build release artifacts for all supported platforms and architectures.
- Build and tag Docker images.
- Run release validation checks.
- Publish checksums.
- Attach artifacts to the GitHub release.
- Publish changelog or release notes.
- Preserve build provenance where possible.

## 3.4 Matrix builds
- Linux.
- Windows.
- macOS.
- Docker.
- Supported CPU architectures.

## 3.5 Pipeline quality gates
- Required tests pass.
- Packaging succeeds.
- Artifact integrity checks succeed.
- Version metadata is correct.
- Docker images are built for documented architectures.

## 3.6 Workflow maintenance
The CI/CD system must be version-controlled and documented so contributors understand what runs on pull requests, merges, tags, and release publication.

# 4. Release management
- FleetPulse must use semantic versioning.
- Breaking API changes require a new major version.
- Minor releases must preserve compatibility within a major line.
- Patch releases must not break documented behavior.
- Each release must be traceable to a Git commit or tag, CI run, build artifact set, and tested schema version.

# 5. Milestones
- PR checks pass for lint, unit, and contract tests.
- Main-branch builds validate packages and Docker images.
- Tagged releases publish signed or checksum-verifiable artifacts.
- Release notes and artifact traceability are consistent.

# 6. Acceptance criteria
- The full product can be validated before release by automated pipelines.
- Every supported target has a test and packaging path.
- Release artifacts can be traced and verified.
