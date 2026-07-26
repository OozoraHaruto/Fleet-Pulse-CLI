#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

export RELEASE_VERSION=v0.0.0-test
export GITHUB_SHA=0000000000000000000000000000000000000000
export DIST_DIR="$tmp_dir/dist"
export GOCACHE="$tmp_dir/go-build-cache"
export GOMODCACHE="$tmp_dir/go-mod-cache"

require_entry() {
  local contents="$1"
  local entry="$2"
  if ! grep -Fxq "$entry" <<<"$contents"; then
    echo "archive missing expected entry: $entry" >&2
    exit 1
  fi
}

reject_prefix() {
  local contents="$1"
  local prefix="$2"
  if grep -Fq "$prefix" <<<"$contents"; then
    echo "archive contains rejected path prefix: $prefix" >&2
    exit 1
  fi
}

check_unix_archive() {
  local goos="$1"
  local goarch="$2"
  local artifact_os="$3"
  local service_entry="$4"
  local package="fleetpulse-${RELEASE_VERSION}-${artifact_os}-${goarch}"
  local archive="$DIST_DIR/${package}.tar.gz"

  GOOS="$goos" GOARCH="$goarch" ARTIFACT_OS="$artifact_os" bash "$repo_root/scripts/build-release-artifact.sh"

  if [ ! -f "$archive" ]; then
    echo "missing release archive: $archive" >&2
    exit 1
  fi

  local contents
  contents="$(tar -tzf "$archive")"

  require_entry "$contents" "$package/fleetpulse"
  require_entry "$contents" "$package/install.sh"
  require_entry "$contents" "$package/uninstall.sh"
  require_entry "$contents" "$package/$service_entry"
  require_entry "$contents" "$package/docs/INSTALLATION.md"

  reject_prefix "$contents" "$package/packaging/"
  reject_prefix "$contents" "$package/linux/"
  reject_prefix "$contents" "$package/macos/"
  reject_prefix "$contents" "$package/windows/"
  reject_prefix "$contents" "$package/docs/superpowers/"
}

check_windows_archive() {
  local package="fleetpulse-${RELEASE_VERSION}-windows-amd64"
  local archive="$DIST_DIR/${package}.zip"

  GOOS=windows GOARCH=amd64 ARTIFACT_OS=windows bash "$repo_root/scripts/build-release-artifact.sh"

  if [ ! -f "$archive" ]; then
    echo "missing release archive: $archive" >&2
    exit 1
  fi

  local contents
  contents="$(zipinfo -1 "$archive")"

  require_entry "$contents" "$package/fleetpulse.exe"
  require_entry "$contents" "$package/install.ps1"
  require_entry "$contents" "$package/uninstall.ps1"
  require_entry "$contents" "$package/docs/INSTALLATION.md"

  reject_prefix "$contents" "$package/packaging/"
  reject_prefix "$contents" "$package/linux/"
  reject_prefix "$contents" "$package/macos/"
  reject_prefix "$contents" "$package/windows/"
  reject_prefix "$contents" "$package/docs/superpowers/"
}

check_unix_archive linux amd64 linux fleetpulse.service
check_unix_archive darwin amd64 macos com.fleetpulse.agent.plist
check_windows_archive
