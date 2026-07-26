# FleetPulse Troubleshooting Guide

## Service Does Not Start

Run:

```sh
fleetpulse diagnose -config /etc/fleetpulse/fleetpulse.json
```

Check whether a non-local bind is configured without authentication. FleetPulse refuses that configuration.

## API Returns 401

Confirm the client sends:

```http
Authorization: Bearer <token>
```

Retrieve the token with `fleetpulse token show` from an admin shell.

## Metrics Are Unsupported Or Unavailable

Unsupported means FleetPulse does not have a collector for that platform or device yet. Unavailable means the collector could not read the value at runtime, often because of permissions or container isolation.

## Docker Replacement Lost Auth

Ensure `/var/lib/fleetpulse` is backed by a persistent volume. Removing the volume intentionally deletes token state.
