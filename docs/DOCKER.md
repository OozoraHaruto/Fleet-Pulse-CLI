# FleetPulse Docker Guide

Build locally:

```sh
docker build --target runtime -t fleetpulse:local .
```

Build the NVIDIA CUDA image locally:

```sh
docker build --target cuda -t fleetpulse:local-cuda .
```

Build the AMD ROCm image locally:

```sh
docker build --target rocm -t fleetpulse:local-rocm .
```

Build the Intel GPU tools image locally:

```sh
docker build --target intel-gpu -t fleetpulse:local-intel-gpu .
```

Run with persistent state:

```sh
docker run --rm \
  -p 35338:35338 \
  -v fleetpulse-state:/var/lib/fleetpulse \
  fleetpulse:local
```

The default Docker command binds to `0.0.0.0:35338` with authentication enabled and stores the token at `/var/lib/fleetpulse/token`.

Release builds publish a standard image plus GPU-tooling variants:

- `ghcr.io/<owner>/<repo>:<version>`
- `ghcr.io/<owner>/<repo>:<version>-cuda`
- `ghcr.io/<owner>/<repo>:<version>-rocm`
- `ghcr.io/<owner>/<repo>:<version>-intel-gpu`

The `-cuda` image is based on NVIDIA's CUDA image and sets the driver capabilities needed for `nvidia-smi`. Run it with NVIDIA GPU access enabled:

```sh
docker run --rm --gpus all --entrypoint nvidia-smi fleetpulse:local-cuda
```

The `-rocm` image is based on AMD's ROCm image. Run it with AMD GPU devices mounted:

```sh
docker run --rm \
  --device=/dev/kfd \
  --device=/dev/dri \
  --group-add video \
  --entrypoint rocm-smi \
  fleetpulse:local-rocm
```

On newer ROCm stacks, `amd-smi list` is also available:

```sh
docker run --rm \
  --device=/dev/kfd \
  --device=/dev/dri \
  --group-add video \
  --entrypoint amd-smi \
  fleetpulse:local-rocm list
```

The `-intel-gpu` image installs `intel-gpu-tools` for `intel_gpu_top`. Run it with the host DRM devices mounted:

```sh
docker run --rm \
  --device=/dev/dri \
  --entrypoint intel_gpu_top \
  fleetpulse:local-intel-gpu -L
```

Run FleetPulse with GPU access:

```sh
docker run --rm --gpus all \
  -p 35338:35338 \
  -v fleetpulse-state:/var/lib/fleetpulse \
  fleetpulse:local-cuda
```

For AMD or Intel GPUs, replace the image with `fleetpulse:local-rocm` or `fleetpulse:local-intel-gpu` and pass the matching device flags shown above.

Use Compose:

```sh
docker compose -f deploy/docker-compose.yml up -d
```

To inspect the token:

```sh
docker compose -f deploy/docker-compose.yml exec fleetpulse fleetpulse token show -token-file /var/lib/fleetpulse/token
```
