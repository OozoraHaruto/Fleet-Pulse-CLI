# FleetPulse Uninstallation Guide

Uninstall preserves config and token state by default so rollback and reinstall do not require re-provisioning.

## Linux

```sh
sudo ./uninstall.sh
```

To remove preserved config, token, and logs:

```sh
sudo PURGE=true ./uninstall.sh
```

From a source checkout, use `packaging/linux/uninstall.sh`.

## macOS

```sh
sudo ./uninstall.sh
```

To purge state:

```sh
sudo PURGE=true ./uninstall.sh
```

From a source checkout, use `packaging/macos/uninstall.sh`.

## Windows

```powershell
.\uninstall.ps1
```

To purge state:

```powershell
.\uninstall.ps1 -Purge
```

From a source checkout, use `.\packaging\windows\uninstall.ps1`.

## Docker

Remove the container to stop FleetPulse. Remove the named volumes only when you intentionally want to delete config and token state.
