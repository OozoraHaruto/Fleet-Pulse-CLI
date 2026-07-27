#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

require_contains() {
  local file="$1"
  local pattern="$2"
  local description="$3"

  if ! grep -Fq -- "$pattern" "$repo_root/$file"; then
    echo "$file missing $description: $pattern" >&2
    exit 1
  fi
}

reject_contains() {
  local file="$1"
  local pattern="$2"
  local description="$3"

  if grep -Fq -- "$pattern" "$repo_root/$file"; then
    echo "$file contains rejected $description: $pattern" >&2
    exit 1
  fi
}

require_contains Dockerfile "FROM nvidia/cuda:11.8.0-base-ubuntu22.04 AS cuda" "driver-compatible NVIDIA CUDA runtime base image"
require_contains Dockerfile "AS cuda" "named CUDA image target"
require_contains Dockerfile "NVIDIA_DRIVER_CAPABILITIES=compute,utility" "nvidia-smi driver capability"
reject_contains Dockerfile "nvidia/cuda:13." "CUDA 13 image with high host-driver requirement"
require_contains Dockerfile "FROM rocm/dev-ubuntu-24.04:" "AMD ROCm runtime base image"
require_contains Dockerfile "AS rocm" "named ROCm image target"
require_contains Dockerfile "FROM ubuntu:24.04 AS intel-gpu" "named Intel GPU image target"
require_contains Dockerfile "intel-gpu-tools" "Intel GPU tooling package"

require_contains .github/workflows/ci.yml "docker_target: runtime" "standard Docker CI matrix target"
require_contains .github/workflows/ci.yml "docker_target: cuda" "CUDA Docker CI matrix target"
require_contains .github/workflows/ci.yml "docker_target: rocm" "ROCm Docker CI matrix target"
require_contains .github/workflows/ci.yml "docker_target: intel-gpu" "Intel GPU Docker CI matrix target"
require_contains .github/workflows/ci.yml '--target "$DOCKER_TARGET"' "CI Docker matrix build command"
require_contains .github/workflows/ci.yml 'name: Docker build ${{ matrix.docker_target }}' "parallel Docker validation job"
require_contains .github/workflows/ci.yml "bash tests/docker-release-variants.sh" "Docker release variant check"
reject_contains .github/workflows/ci.yml "fleetpulse:ci-rocm" "sequential ROCm CI Docker validation"
reject_contains .github/workflows/ci.yml "fleetpulse:ci-intel-gpu" "sequential Intel GPU CI Docker validation"

require_contains .github/workflows/release.yml "docker_target: runtime" "standard Docker release matrix target"
require_contains .github/workflows/release.yml "docker_target: cuda" "CUDA Docker release matrix target"
require_contains .github/workflows/release.yml "docker_target: rocm" "ROCm Docker release matrix target"
require_contains .github/workflows/release.yml "docker_target: intel-gpu" "Intel GPU Docker release matrix target"
require_contains .github/workflows/release.yml 'name: Docker build check ${{ matrix.docker_target }}' "parallel Docker release validation job"
require_contains .github/workflows/release.yml 'IMAGE_SUFFIX: ${{ matrix.image_suffix }}' "Docker release matrix suffix env"
require_contains .github/workflows/release.yml '--target "$DOCKER_TARGET"' "Docker release target build"
require_contains .github/workflows/release.yml ':${RELEASE_VERSION}${IMAGE_SUFFIX}' "Docker release variant image tag"
require_contains .github/workflows/release.yml "bash tests/docker-release-variants.sh" "Docker release variant check"
reject_contains .github/workflows/release.yml "fleetpulse:release-check-rocm" "sequential ROCm release Docker validation"
reject_contains .github/workflows/release.yml "fleetpulse:release-check-intel-gpu" "sequential Intel GPU release Docker validation"

require_contains docs/DOCKER.md "-cuda" "CUDA image documentation"
require_contains docs/DOCKER.md "nvidia-smi" "NVIDIA GPU smoke test documentation"
require_contains docs/DOCKER.md "-rocm" "ROCm image documentation"
require_contains docs/DOCKER.md "rocm-smi" "AMD GPU smoke test documentation"
require_contains docs/DOCKER.md "-intel-gpu" "Intel GPU image documentation"
require_contains docs/DOCKER.md "intel_gpu_top" "Intel GPU smoke test documentation"
require_contains docs/RELEASE.md "-cuda" "CUDA release variant documentation"
require_contains docs/RELEASE.md "-rocm" "ROCm release variant documentation"
require_contains docs/RELEASE.md "-intel-gpu" "Intel GPU release variant documentation"
