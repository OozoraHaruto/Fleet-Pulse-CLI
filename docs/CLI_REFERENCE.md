# FleetPulse CLI Reference

## Serve

```sh
fleetpulse serve -addr 127.0.0.1:8080
```

Important flags:

- `-config`: JSON config file path.
- `-addr`: bind address and port.
- `-auth`: enable bearer-token authentication.
- `-token-file`: token file path.
- `-cache-ttl`: snapshot cache TTL.
- `-collector-timeout`: collector timeout.
- `-deployment-target`: diagnostic target override.

## Token

```sh
fleetpulse token show -token-file /var/lib/fleetpulse/token
fleetpulse token rotate -token-file /var/lib/fleetpulse/token
```

`show` prints the current token. `rotate` replaces it and prints the new token once.

## Diagnostics

```sh
fleetpulse diagnose -config /etc/fleetpulse/fleetpulse.json
```

Diagnostics include config paths, bind safety, collector settings, and token-file presence. They do not print token values.

## Config

```sh
fleetpulse config show -config /etc/fleetpulse/fleetpulse.json
```

This prints the effective configuration after file and environment handling.
