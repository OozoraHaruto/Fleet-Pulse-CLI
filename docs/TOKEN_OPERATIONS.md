# FleetPulse Token Operations

FleetPulse uses a static bearer token for the initial authentication mechanism.

## Provisioning

On first authenticated startup, FleetPulse creates the token file if it does not exist and prints the initial token once. Installers also generate a token when one is missing.

## Show

```sh
fleetpulse token show -token-file /var/lib/fleetpulse/token
```

Run this as an administrator or service owner because the token file is created with protected permissions.

## Rotate

```sh
fleetpulse token rotate -token-file /var/lib/fleetpulse/token
```

Rotation replaces the token file and prints the new token once. Existing clients must update their `Authorization` header.

## Upgrade Behavior

Native and Docker upgrades preserve token files by default. Only explicit purge or volume deletion removes token state.
