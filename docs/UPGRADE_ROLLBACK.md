# FleetPulse Upgrade and Rollback Guide

## Upgrade

Installers overwrite the binary and service definition, then restart the service when requested. They preserve config and token files by default.

Docker upgrades replace the image while preserving mounted config and state volumes:

```sh
docker compose -f deploy/docker-compose.yml pull
docker compose -f deploy/docker-compose.yml up -d
```

## Rollback

Reinstall the previous binary or redeploy the previous Docker image using the same config and state paths. Because token state is preserved, clients do not need new credentials after rollback.

## Purge

Only use purge options when intentionally deleting FleetPulse state:

- Linux/macOS: `PURGE=true`
- Windows: `-Purge`
- Docker: delete the named volumes
