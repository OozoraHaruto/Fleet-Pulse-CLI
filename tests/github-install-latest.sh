#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

version="v9.8.7"
artifact_os="macos"
goarch="arm64"
package="fleetpulse-${version}-${artifact_os}-${goarch}"
archive_root="$tmp_dir/archive-root"
release_dir="$tmp_dir/release"
bin_dir="$tmp_dir/bin"

mkdir -p "$archive_root/$package" "$release_dir" "$bin_dir"

printf '#!/usr/bin/env sh\nexit 0\n' >"$archive_root/$package/fleetpulse"
chmod 0755 "$archive_root/$package/fleetpulse"

cat >"$archive_root/$package/install.sh" <<'INSTALLER'
#!/usr/bin/env sh
set -eu
printf '%s\n' "$1" >"$FLEETPULSE_TEST_INVOKED"
test -x "$1"
INSTALLER
chmod 0755 "$archive_root/$package/install.sh"

tar -czf "$release_dir/${package}.tar.gz" -C "$archive_root" "$package"
(
  cd "$release_dir"
  sha256sum "${package}.tar.gz" >"${package}.tar.gz.sha256"
)

cat >"$release_dir/latest.json" <<JSON
{
  "tag_name": "$version"
}
JSON

cat >"$bin_dir/curl" <<'CURL'
#!/usr/bin/env bash
set -euo pipefail

out=""
url=""

while (($#)); do
  case "$1" in
    -o)
      shift
      out="$1"
      ;;
    -H)
      shift
      ;;
    -*)
      ;;
    *)
      url="$1"
      ;;
  esac
  shift || true
done

case "$url" in
  https://api.github.com/repos/OozoraHaruto/Fleet-Pulse-CLI/releases/latest)
    src="$FLEETPULSE_TEST_RELEASE_DIR/latest.json"
    ;;
  https://github.com/OozoraHaruto/Fleet-Pulse-CLI/releases/download/v9.8.7/fleetpulse-v9.8.7-macos-arm64.tar.gz)
    src="$FLEETPULSE_TEST_RELEASE_DIR/fleetpulse-v9.8.7-macos-arm64.tar.gz"
    ;;
  https://github.com/OozoraHaruto/Fleet-Pulse-CLI/releases/download/v9.8.7/fleetpulse-v9.8.7-macos-arm64.tar.gz.sha256)
    src="$FLEETPULSE_TEST_RELEASE_DIR/fleetpulse-v9.8.7-macos-arm64.tar.gz.sha256"
    ;;
  *)
    echo "unexpected curl URL: $url" >&2
    exit 22
    ;;
esac

if [ -n "$out" ]; then
  cp "$src" "$out"
else
  cat "$src"
fi
CURL
chmod 0755 "$bin_dir/curl"

invoked_file="$tmp_dir/invoked"

PATH="$bin_dir:$PATH" \
  FLEETPULSE_INSTALL_OS="$artifact_os" \
  FLEETPULSE_INSTALL_ARCH="$goarch" \
  FLEETPULSE_USE_SUDO=false \
  FLEETPULSE_TEST_RELEASE_DIR="$release_dir" \
  FLEETPULSE_TEST_INVOKED="$invoked_file" \
  sh "$repo_root/install.sh"

invoked="$(cat "$invoked_file")"
case "$invoked" in
  */"$package"/fleetpulse) ;;
  *)
    echo "installer invoked with unexpected binary path: $invoked" >&2
    exit 1
    ;;
esac
