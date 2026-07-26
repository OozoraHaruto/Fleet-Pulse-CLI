# FleetPulse Uninstallation Guide

Uninstall preserves config and token state by default so rollback and reinstall do not require re-provisioning.

## Linux

```sh
sudo packaging/linux/uninstall.sh
```

To remove preserved config, token, and logs:

```sh
sudo PURGE=true packaging/linux/uninstall.sh
```

## macOS

```sh
sudo packaging/macos/uninstall.sh
```

To purge state:

```sh
sudo PURGE=true packaging/macos/uninstall.sh
```

## Windows

```powershell
.\packaging\windows\uninstall.ps1
```

To purge state:

```powershell
.\packaging\windows\uninstall.ps1 -Purge
```

## Docker

Remove the container to stop FleetPulse. Remove the named volumes only when you intentionally want to delete config and token state.
