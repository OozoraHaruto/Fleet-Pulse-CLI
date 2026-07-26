# FleetPulse Docker Guide

Build locally:

```sh
docker build -t fleetpulse:local .
```

Run with persistent state:

```sh
docker run --rm \
  -p 35338:35338 \
  -v fleetpulse-state:/var/lib/fleetpulse \
  fleetpulse:local
```

The default Docker command binds to `0.0.0.0:35338` with authentication enabled and stores the token at `/var/lib/fleetpulse/token`.

Use Compose:

```sh
docker compose -f deploy/docker-compose.yml up -d
```

To inspect the token:

```sh
docker compose -f deploy/docker-compose.yml exec fleetpulse fleetpulse token show -token-file /var/lib/fleetpulse/token
```
