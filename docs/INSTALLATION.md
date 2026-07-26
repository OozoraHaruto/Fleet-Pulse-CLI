# FleetPulse Installation Guide

FleetPulse ships as a single binary plus platform lifecycle artifacts.

## Linux

Build or download `fleetpulse`, then run:

```sh
sudo packaging/linux/install.sh ./fleetpulse
```

The installer copies the binary to `/usr/local/bin/fleetpulse`, creates `/etc/fleetpulse` and `/var/lib/fleetpulse`, installs `fleetpulse.service`, enables the service, and starts it unless `START_SERVICE=false` is set.

## macOS

```sh
sudo packaging/macos/install.sh ./fleetpulse
```

The installer copies the binary to `/usr/local/bin/fleetpulse`, writes config/state under `/Library/Application Support/FleetPulse`, installs a launch daemon, and starts it unless `START_SERVICE=false` is set.

## Windows

Run PowerShell as Administrator:

```powershell
.\packaging\windows\install.ps1 -BinarySource .\fleetpulse.exe -StartService
```

The installer copies the binary under `Program Files`, writes config/state under `ProgramData`, and registers the Windows service.

## Docker

Use the provided `Dockerfile` or `deploy/docker-compose.yml`. Docker deployments should mount `/var/lib/fleetpulse` so token state survives image replacement.
