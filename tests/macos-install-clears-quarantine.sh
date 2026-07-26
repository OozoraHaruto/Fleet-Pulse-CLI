#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

mkdir -p "$tmp_dir/bin" "$tmp_dir/config" "$tmp_dir/state"

xattr_log="$tmp_dir/xattr.log"
cat >"$tmp_dir/bin/xattr" <<'EOF'
#!/usr/bin/env sh
printf '%s\n' "$*" >> "$XATTR_LOG"
EOF
chmod +x "$tmp_dir/bin/xattr"

cat >"$tmp_dir/bin/install" <<'EOF'
#!/usr/bin/env sh
mode=""
if [ "${1:-}" = "-d" ]; then
  shift
  while [ "$#" -gt 0 ]; do
    case "$1" in
      -m)
        shift 2
        ;;
      *)
        mkdir -p "$1"
        shift
        ;;
    esac
  done
  exit 0
fi

while [ "$#" -gt 0 ]; do
  case "$1" in
    -m)
      mode="$2"
      shift 2
      ;;
    *)
      break
      ;;
  esac
done

src="$1"
dest="$2"
case "$dest" in
  /Library/LaunchDaemons/*)
    dest="$FLEETPULSE_TEST_INSTALL_ROOT/$(basename "$dest")"
    ;;
esac

mkdir -p "$(dirname "$dest")"
cp "$src" "$dest"
if [ -n "$mode" ]; then
  chmod "$mode" "$dest"
fi
EOF
chmod +x "$tmp_dir/bin/install"

binary_source="$tmp_dir/fleetpulse-source"
cat >"$binary_source" <<'EOF'
#!/usr/bin/env sh
token_file=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -token-file)
      token_file="$2"
      shift 2
      ;;
    *)
      break
      ;;
  esac
done

case "${1:-} ${2:-}" in
  "token show")
    exit 1
    ;;
  "token rotate")
    while [ "$#" -gt 0 ]; do
      case "$1" in
        -token-file)
          token_file="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [ -z "$token_file" ]; then
      echo "missing -token-file" >&2
      exit 2
    fi
    printf 'test-token\n' >"$token_file"
    exit 0
    ;;
  *)
    echo "unexpected fleetpulse command: $*" >&2
    exit 2
    ;;
esac
EOF
chmod +x "$binary_source"

PATH="$tmp_dir/bin:$PATH" \
XATTR_LOG="$xattr_log" \
FLEETPULSE_TEST_INSTALL_ROOT="$tmp_dir/launchdaemons" \
PREFIX="$tmp_dir/prefix" \
CONFIG_DIR="$tmp_dir/config" \
STATE_DIR="$tmp_dir/state" \
START_SERVICE=false \
  sh "$repo_root/packaging/macos/install.sh" "$binary_source" >/dev/null

if ! grep -Fxq -- "-d com.apple.quarantine $binary_source" "$xattr_log"; then
  echo "installer did not clear quarantine on source binary" >&2
  exit 1
fi

if ! grep -Fxq -- "-d com.apple.quarantine $tmp_dir/prefix/bin/fleetpulse" "$xattr_log"; then
  echo "installer did not clear quarantine on installed binary" >&2
  exit 1
fi
