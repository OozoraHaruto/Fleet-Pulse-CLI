# FleetPulse Docker Guide

Build locally:

```sh
docker build --target runtime -t fleetpulse:local .
```

Build the NVIDIA CUDA image locally:

```sh
docker build --target cuda -t fleetpulse:local-cuda .
```

Run with persistent state:

```sh
docker run --rm \
  -p 35338:35338 \
  -v fleetpulse-state:/var/lib/fleetpulse \
  fleetpulse:local
```

The default Docker command binds to `0.0.0.0:35338` with authentication enabled and stores the token at `/var/lib/fleetpulse/token`.

Release builds publish a standard image and an NVIDIA CUDA variant:

- `ghcr.io/<owner>/<repo>:<version>`
- `ghcr.io/<owner>/<repo>:<version>-cuda`

The `-cuda` image is based on NVIDIA's CUDA image and sets the driver capabilities needed for `nvidia-smi`. Run it with NVIDIA GPU access enabled:

```sh
docker run --rm --gpus all --entrypoint nvidia-smi fleetpulse:local-cuda
```

Run FleetPulse with GPU access:

```sh
docker run --rm --gpus all \
  -p 35338:35338 \
  -v fleetpulse-state:/var/lib/fleetpulse \
  fleetpulse:local-cuda
```

Use Compose:

```sh
docker compose -f deploy/docker-compose.yml up -d
```

To inspect the token:

```sh
docker compose -f deploy/docker-compose.yml exec fleetpulse fleetpulse token show -token-file /var/lib/fleetpulse/token
```
