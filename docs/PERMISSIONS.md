# FleetPulse Permissions Guide

FleetPulse is read-only. Native services should run with the lowest practical privileges.

## Linux

The systemd unit runs as the `fleetpulse` user, uses `NoNewPrivileges=true`, protects home directories, and grants write access only to FleetPulse state/log paths.

## macOS

The launch daemon runs the installed binary with config and token state under `/Library/Application Support/FleetPulse`.

## Windows

The service runs with the account selected by Windows service configuration. Restrict access to `ProgramData\FleetPulse` to administrators and the service identity.

## Docker

The container runs as an unprivileged user. Host-level disk, GPU, or hardware health visibility may require extra mounts, device mappings, or capabilities. Missing permission is reported as unavailable rather than crashing the API.
