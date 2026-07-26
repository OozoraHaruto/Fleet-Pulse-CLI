# FleetPulse PRD 3

## Installation, Uninstallation, and Packaging

# 1. Product scope
This PRD defines how FleetPulse is installed, upgraded, packaged, and removed across native hosts and Docker.

# 2. Installation and uninstallation requirements
- FleetPulse must include first-class installation, upgrade, and uninstallation support for every deployment target.
- FleetPulse must provide install and uninstall support for Linux, Windows, and macOS.
- FleetPulse must support Docker-based deployment through documented runtime and volume patterns.

# 3. Linux installation
- A Linux install script must be provided.
- A systemd service file must be provided.
- A systemd install workflow must be provided.
- A systemd uninstall workflow must be provided.
- The installer must install the binary, create config/data directories, create and enable the service, and start it unless disabled.
- The installer must preserve or generate the auth token according to the provisioning rules.
- Uninstall must support clean removal of the service and installed files.

# 4. Windows installation
- A Windows installer or install script must be provided.
- Windows service registration must be provided.
- Windows service uninstall support must be provided.
- The installer must install the binary, register the service, start it unless disabled, and persist the auth token across upgrades.
- Uninstall must remove the service and application files according to operator choice.

# 5. macOS installation
- A macOS install script or installer package must be provided.
- Launch daemon or equivalent background service configuration must be provided.
- Uninstall support must be provided.
- The installer must place the binary correctly, install and enable the background service, start it unless disabled, and preserve the auth token across upgrades.

# 6. Docker installation
- A documented Docker image must be provided.
- Container run examples must be provided.
- Optional Docker Compose examples must be provided.
- Volume-based persistence for config and token storage must be supported.
- Docker upgrades must allow image replacement without losing token or config.
- Explicit cleanup/uninstall must be supported by removing the container and associated volumes when requested.

# 7. Required lifecycle scripts
- install
- uninstall
- upgrade
- start
- stop
- status
- token provisioning
- token rotation
- diagnostics
These scripts must be deterministic and documented for each supported deployment target.

# 8. Packaging requirements
- Linux: shell install script and systemd unit package.
- Windows: installer or service setup script.
- macOS: install script or package with launch daemon support.
- Docker: image plus volume-backed runtime setup.
Packaging must make it easy to install, start, stop, uninstall, upgrade, rotate credentials, and preserve state across upgrades.

# 9. Supported architectures
- Linux: amd64, arm64.
- Windows: amd64.
- macOS: amd64, arm64.
- Docker images: amd64, arm64.
Architecture support must be documented per release. Unsupported combinations must be stated clearly.

# 10. Packaging integrity and release management
- Each release should produce native binaries or installers, Docker images, checksum files, and release notes.
- Release artifacts must be verifiable through checksums, signatures, or image digests.
- FleetPulse must use semantic versioning.
- Each release should be traceable to a Git commit or tag, a CI run, a build artifact set, and the tested API schema version.

# 11. Documentation and examples requirements
- Installation guide for each platform.
- Uninstall guide for each platform.
- CLI command reference.
- Configuration reference.
- API reference.
- Docker run examples.
- Docker Compose examples.
- Token provisioning and rotation guide.
- Troubleshooting guide.
- Permissions guide.
- Disk health behavior guide.
- Upgrade and rollback guide.

# 12. Acceptance criteria
- The product can be installed and uninstalled cleanly on Linux, Windows, macOS, and Docker.
- Upgrades preserve config and tokens by default.
- Release artifacts are reproducible and traceable.
- Supported architectures are documented and validated.
