#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

reject_contains() {
  local file="$1"
  local pattern="$2"
  local description="$3"

  if grep -Fq -- "$pattern" "$repo_root/$file"; then
    echo "$file contains rejected $description: $pattern" >&2
    exit 1
  fi
}

require_contains() {
  local file="$1"
  local pattern="$2"
  local description="$3"

  if ! grep -Fq -- "$pattern" "$repo_root/$file"; then
    echo "$file missing $description: $pattern" >&2
    exit 1
  fi
}

for workflow in .github/workflows/ci.yml .github/workflows/release.yml; do
  reject_contains "$workflow" "actions/checkout@v4" "Node 20 checkout action"
  reject_contains "$workflow" "actions/setup-go@v5" "Node 20 setup-go action"
  reject_contains "$workflow" "actions/upload-artifact@v4" "Node 20 upload-artifact action"
  reject_contains "$workflow" "actions/download-artifact@v4" "Node 20 download-artifact action"
  reject_contains "$workflow" "docker/login-action@v3" "Node 20 Docker login action"
  reject_contains "$workflow" "softprops/action-gh-release@v2" "Node 20 GitHub release action"
done

require_contains .github/workflows/ci.yml "actions/checkout@v6" "Node 24 checkout action"
require_contains .github/workflows/ci.yml "actions/setup-go@v6" "Node 24 setup-go action"
require_contains .github/workflows/ci.yml "actions/upload-artifact@v7" "Node 24 upload-artifact action"
require_contains .github/workflows/release.yml "actions/download-artifact@v7" "Node 24 download-artifact action"
require_contains .github/workflows/release.yml "docker/login-action@v4" "Node 24 Docker login action"
require_contains .github/workflows/release.yml "softprops/action-gh-release@v3" "Node 24 GitHub release action"
require_contains .github/workflows/ci.yml "cache-dependency-path: go.mod" "setup-go cache dependency path"
require_contains .github/workflows/release.yml "cache-dependency-path: go.mod" "setup-go cache dependency path"
