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

require_contains Dockerfile "FROM nvidia/cuda:" "NVIDIA CUDA runtime base image"
require_contains Dockerfile "AS cuda" "named CUDA image target"
require_contains Dockerfile "NVIDIA_DRIVER_CAPABILITIES=compute,utility" "nvidia-smi driver capability"

require_contains .github/workflows/ci.yml "--target runtime" "standard Docker target validation"
require_contains .github/workflows/ci.yml "--target cuda" "CUDA Docker target validation"
require_contains .github/workflows/ci.yml "bash tests/docker-release-variants.sh" "Docker release variant check"

require_contains .github/workflows/release.yml "docker_target: runtime" "standard Docker release matrix target"
require_contains .github/workflows/release.yml "docker_target: cuda" "CUDA Docker release matrix target"
require_contains .github/workflows/release.yml 'IMAGE_SUFFIX: ${{ matrix.image_suffix }}' "Docker release matrix suffix env"
require_contains .github/workflows/release.yml '--target "$DOCKER_TARGET"' "Docker release target build"
require_contains .github/workflows/release.yml ':${RELEASE_VERSION}${IMAGE_SUFFIX}' "Docker release variant image tag"
require_contains .github/workflows/release.yml "bash tests/docker-release-variants.sh" "Docker release variant check"

require_contains docs/DOCKER.md "-cuda" "CUDA image documentation"
require_contains docs/DOCKER.md "nvidia-smi" "NVIDIA GPU smoke test documentation"
require_contains docs/RELEASE.md "-cuda" "CUDA release variant documentation"
