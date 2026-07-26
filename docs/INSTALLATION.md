# FleetPulse Installation Guide

FleetPulse ships as a single binary plus platform lifecycle artifacts.

## Latest GitHub Release

On Linux and macOS, install the latest published release from GitHub:

```sh
curl -fsSL https://raw.githubusercontent.com/OozoraHaruto/Fleet-Pulse-CLI/main/install.sh | sh
```

The bootstrap installer detects the local OS and architecture, downloads the matching release archive and `.sha256` file from GitHub Releases, verifies the checksum, unpacks the archive, and runs the bundled platform installer. It uses `sudo` automatically when the current user is not root.

GitHub's latest release API returns only stable releases. To test an early prerelease before publishing a stable release, opt in explicitly:

```sh
curl -fsSL https://raw.githubusercontent.com/OozoraHaruto/Fleet-Pulse-CLI/main/install.sh | FLEETPULSE_ALLOW_PRERELEASE=true sh
```

Set `FLEETPULSE_VERSION` to install a specific release tag:

```sh
curl -fsSL https://raw.githubusercontent.com/OozoraHaruto/Fleet-Pulse-CLI/main/install.sh | FLEETPULSE_VERSION=v1.2.3 sh
```

Set `START_SERVICE=false` to install without starting the service.

If the shell reports `getcwd: cannot access parent directories`, change to a readable directory such as `$HOME` and rerun the command.

## Linux

From a release archive, run:

```sh
sudo ./install.sh ./fleetpulse
```

From a source checkout, use `sudo packaging/linux/install.sh ./fleetpulse`.

The installer copies the binary to `/usr/local/bin/fleetpulse`, creates `/etc/fleetpulse` and `/var/lib/fleetpulse`, installs `fleetpulse.service`, enables the service, and starts it unless `START_SERVICE=false` is set.

## macOS

From a release archive, run:

```sh
sudo ./install.sh ./fleetpulse
```

From a source checkout, use `sudo packaging/macos/install.sh ./fleetpulse`.

The installer copies the binary to `/usr/local/bin/fleetpulse`, writes config/state under `/Library/Application Support/FleetPulse`, installs a launch daemon, and starts it unless `START_SERVICE=false` is set.

## Windows

Run PowerShell as Administrator:

```powershell
.\install.ps1 -BinarySource .\fleetpulse.exe -StartService
```

From a source checkout, use `.\packaging\windows\install.ps1 -BinarySource .\fleetpulse.exe -StartService`.

The installer copies the binary under `Program Files`, writes config/state under `ProgramData`, and registers the Windows service.

## Docker

Use the provided `Dockerfile` or `deploy/docker-compose.yml`. Docker deployments should mount `/var/lib/fleetpulse` so token state survives image replacement.
