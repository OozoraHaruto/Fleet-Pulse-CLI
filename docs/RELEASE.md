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
- Packaging scripts and service templates.
- Docker image or image digest.
- SHA-256 checksum file.
- Release notes.
- Git tag, commit SHA, CI run, and schema version traceability.

## Release Automation

The `Release` GitHub Actions workflow can be started manually with `workflow_dispatch` by entering a version such as `v1.2.3`. It also runs automatically when a matching semver tag like `v1.2.3` is pushed.

Normal pushes and pull requests run CI only. They do not publish releases.

## Compatibility

Breaking API changes require a new major version. Minor and patch releases within a major line must preserve documented v1 behavior.
