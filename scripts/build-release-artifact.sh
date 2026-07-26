#!/usr/bin/env bash
set -euo pipefail

: "${GOOS:?GOOS is required}"
: "${GOARCH:?GOARCH is required}"
: "${ARTIFACT_OS:?ARTIFACT_OS is required}"
: "${RELEASE_VERSION:?RELEASE_VERSION is required}"
: "${GITHUB_SHA:?GITHUB_SHA is required}"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
dist_dir="${DIST_DIR:-dist}"

mkdir -p "$dist_dir"
dist_abs="$(cd "$dist_dir" && pwd -P)"

suffix=""
if [ "$GOOS" = "windows" ]; then
  suffix=".exe"
fi

package="fleetpulse-${RELEASE_VERSION}-${ARTIFACT_OS}-${GOARCH}"
package_dir="$dist_abs/$package"

rm -rf "$package_dir"
mkdir -p "$package_dir"

(
  cd "$repo_root"
  CGO_ENABLED="${CGO_ENABLED:-0}" go build -trimpath -ldflags="-s -w -X main.version=${RELEASE_VERSION} -X main.commit=${GITHUB_SHA}" -o "$package_dir/fleetpulse${suffix}" ./cmd/fleetpulse

  cp -R docs "$package_dir/docs"
  rm -rf "$package_dir/docs/superpowers"
  cp Dockerfile "$package_dir/Dockerfile"

  case "$ARTIFACT_OS" in
    linux)
      cp packaging/linux/* "$package_dir/"
      ;;
    macos)
      cp packaging/macos/* "$package_dir/"
      ;;
    windows)
      cp packaging/windows/* "$package_dir/"
      ;;
    *)
      echo "unsupported ARTIFACT_OS: $ARTIFACT_OS" >&2
      exit 1
      ;;
  esac
)

if [ "$GOOS" = "windows" ]; then
  (
    cd "$dist_abs"
    rm -f "${package}.zip" "${package}.zip.sha256"
    zip -qr "${package}.zip" "$package"
    sha256sum "${package}.zip" > "${package}.zip.sha256"
  )
else
  (
    cd "$dist_abs"
    rm -f "${package}.tar.gz" "${package}.tar.gz.sha256"
    tar -czf "${package}.tar.gz" "$package"
    sha256sum "${package}.tar.gz" > "${package}.tar.gz.sha256"
  )
fi
