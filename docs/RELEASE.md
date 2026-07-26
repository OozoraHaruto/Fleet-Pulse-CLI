# FleetPulse Release Guide

FleetPulse uses semantic versioning.

## Supported Targets

- Linux: amd64, arm64
- Windows: amd64
- macOS: amd64, arm64
- Docker: amd64, arm64

## Release Contents

Each release should include:

- Native binaries.
- Platform-specific packaging scripts and service templates at the archive root.
- Operator documentation, excluding internal `docs/superpowers` planning artifacts.
- Docker image or image digest.
- SHA-256 checksum file.
- Release notes.
- Git tag, commit SHA, CI run, and schema version traceability.

## Release Automation

`version.txt` is the stable release source of truth. It must contain a final semantic version such as `v1.0.0`.

The CI workflow resolves artifact versions from `origin/main:version.txt` when that ref is available, then falls back to the checked-out `version.txt`. CI artifact builds always append a unique prerelease suffix in the form `-ci.<run>.<sha>`.

The `Release` GitHub Actions workflow runs automatically for pushes to `main` and publishes a GitHub prerelease using the same `-ci.<run>.<sha>` suffix. It also runs for matching semver tags like `v1.2.3` and can be started manually with `workflow_dispatch`; those final releases use the exact value in `version.txt` and are not marked as prereleases. Tag releases fail if the pushed tag does not match `version.txt`.

Before building artifacts, pushing Docker images, deleting/replacing an existing CI prerelease, or publishing a GitHub release, the release workflow runs formatting, Go tests, binary build, packaging script checks, release layout checks, installer tests, and Docker build validation.

Pull requests run CI only. They do not publish GitHub releases.

## Compatibility

Breaking API changes require a new major version. Minor and patch releases within a major line must preserve documented v1 behavior.
