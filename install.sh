#!/usr/bin/env sh
set -eu

REPO="${FLEETPULSE_REPO:-OozoraHaruto/Fleet-Pulse-CLI}"
VERSION="${FLEETPULSE_VERSION:-latest}"
USE_SUDO="${FLEETPULSE_USE_SUDO:-auto}"

fail() {
  echo "fleetpulse install: $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

github_api_get() {
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    curl -fsSL \
      -H "Accept: application/vnd.github+json" \
      -H "Authorization: Bearer $GITHUB_TOKEN" \
      "$@"
  else
    curl -fsSL \
      -H "Accept: application/vnd.github+json" \
      "$@"
  fi
}

github_download() {
  url="$1"
  dest="$2"

  if [ -n "${GITHUB_TOKEN:-}" ]; then
    curl -fsSL \
      -H "Authorization: Bearer $GITHUB_TOKEN" \
      -o "$dest" \
      "$url"
  else
    curl -fsSL \
      -o "$dest" \
      "$url"
  fi
}

detect_os() {
  if [ -n "${FLEETPULSE_INSTALL_OS:-}" ]; then
    printf '%s\n' "$FLEETPULSE_INSTALL_OS"
    return
  fi

  case "$(uname -s)" in
    Darwin)
      printf 'macos\n'
      ;;
    Linux)
      printf 'linux\n'
      ;;
    *)
      fail "unsupported OS: $(uname -s). Windows installs use packaging/windows/install.ps1."
      ;;
  esac
}

detect_arch() {
  if [ -n "${FLEETPULSE_INSTALL_ARCH:-}" ]; then
    printf '%s\n' "$FLEETPULSE_INSTALL_ARCH"
    return
  fi

  case "$(uname -m)" in
    x86_64 | amd64)
      printf 'amd64\n'
      ;;
    arm64 | aarch64)
      printf 'arm64\n'
      ;;
    *)
      fail "unsupported architecture: $(uname -m)"
      ;;
  esac
}

resolve_version() {
  if [ "$VERSION" != "latest" ]; then
    printf '%s\n' "$VERSION"
    return
  fi

  latest_json="$(github_api_get "https://api.github.com/repos/$REPO/releases/latest")" ||
    fail "could not fetch latest release metadata from GitHub"
  latest_version="$(printf '%s\n' "$latest_json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"

  if [ -z "$latest_version" ]; then
    fail "could not find tag_name in latest GitHub release metadata"
  fi

  printf '%s\n' "$latest_version"
}

verify_checksum() {
  checksum_dir="$1"
  checksum_file="$2"

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$checksum_dir" && sha256sum -c "$checksum_file") >/dev/null
    return
  fi

  if command -v shasum >/dev/null 2>&1; then
    (cd "$checksum_dir" && shasum -a 256 -c "$checksum_file") >/dev/null
    return
  fi

  fail "required checksum command not found: sha256sum or shasum"
}

run_platform_installer() {
  installer="$1"
  binary="$2"

  case "$USE_SUDO" in
    false | no | 0)
      sh "$installer" "$binary"
      ;;
    true | yes | 1)
      if [ "$(id -u)" = "0" ]; then
        sh "$installer" "$binary"
      else
        require_cmd sudo
        sudo -E sh "$installer" "$binary"
      fi
      ;;
    auto)
      if [ "$(id -u)" = "0" ]; then
        sh "$installer" "$binary"
      else
        require_cmd sudo
        sudo -E sh "$installer" "$binary"
      fi
      ;;
    *)
      fail "FLEETPULSE_USE_SUDO must be auto, true, or false"
      ;;
  esac
}

require_cmd curl
require_cmd tar

artifact_os="$(detect_os)"
artifact_arch="$(detect_arch)"
release_version="$(resolve_version)"

case "$artifact_os" in
  linux | macos) ;;
  windows)
    fail "Windows installs use packaging/windows/install.ps1"
    ;;
  *)
    fail "unsupported release OS: $artifact_os"
    ;;
esac

package="fleetpulse-${release_version}-${artifact_os}-${artifact_arch}"
archive_name="${package}.tar.gz"
archive_url="https://github.com/$REPO/releases/download/$release_version/$archive_name"
checksum_url="${archive_url}.sha256"

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/fleetpulse-install.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT

echo "Installing FleetPulse $release_version for $artifact_os/$artifact_arch from $REPO..."

github_download "$archive_url" "$tmp_dir/$archive_name" ||
  fail "could not download release archive: $archive_url"
github_download "$checksum_url" "$tmp_dir/$archive_name.sha256" ||
  fail "could not download checksum: $checksum_url"

verify_checksum "$tmp_dir" "$archive_name.sha256" ||
  fail "checksum verification failed for $archive_name"

tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"

binary="$tmp_dir/$package/fleetpulse"
installer="$tmp_dir/$package/install.sh"

if [ ! -x "$binary" ]; then
  fail "release archive did not contain executable binary: $package/fleetpulse"
fi

if [ ! -f "$installer" ]; then
  fail "release archive did not contain platform installer: $package/install.sh"
fi

run_platform_installer "$installer" "$binary"

echo "FleetPulse $release_version installed."
