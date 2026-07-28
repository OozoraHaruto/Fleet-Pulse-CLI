#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

mkdir -p "$tmp_dir/bin" "$tmp_dir/config" "$tmp_dir/state" "$tmp_dir/log" "$tmp_dir/systemd"
printf 'existing-token\n' >"$tmp_dir/state/token"

cat >"$tmp_dir/config/fleetpulse.json" <<'JSON'
{
  "addr": "127.0.0.1:35338",
  "auth_enabled": false,
  "token_file": "/custom/token",
  "cache_ttl": "10s",
  "collector_timeout": "2s",
  "log_level": "info",
  "service_name": "fleetpulse",
  "deployment_target": "linux"
}
JSON

cat >"$tmp_dir/bin/install" <<'EOF'
#!/usr/bin/env sh
if [ "${1:-}" = "-d" ]; then
  shift
  while [ "$#" -gt 0 ]; do
    case "$1" in
      -m|-o|-g)
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

mode=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -m)
      mode="$2"
      shift 2
      ;;
    -o|-g)
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
  /etc/systemd/system/fleetpulse.service)
    dest="$FLEETPULSE_TEST_SYSTEMD/fleetpulse.service"
    ;;
esac

mkdir -p "$(dirname "$dest")"
cp "$src" "$dest"
if [ -n "$mode" ]; then
  chmod "$mode" "$dest"
fi
EOF
chmod +x "$tmp_dir/bin/install"

cat >"$tmp_dir/bin/id" <<'EOF'
#!/usr/bin/env sh
exit 0
EOF
chmod +x "$tmp_dir/bin/id"

cat >"$tmp_dir/bin/chown" <<'EOF'
#!/usr/bin/env sh
exit 0
EOF
chmod +x "$tmp_dir/bin/chown"

cat >"$tmp_dir/bin/systemctl" <<'EOF'
#!/usr/bin/env sh
printf '%s\n' "$*" >> "$FLEETPULSE_TEST_SYSTEMCTL_LOG"
exit 0
EOF
chmod +x "$tmp_dir/bin/systemctl"

binary_source="$tmp_dir/fleetpulse-source"
cat >"$binary_source" <<'EOF'
#!/usr/bin/env sh
case "${1:-} ${2:-}" in
  "token show")
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
FLEETPULSE_TEST_SYSTEMD="$tmp_dir/systemd" \
FLEETPULSE_TEST_SYSTEMCTL_LOG="$tmp_dir/systemctl.log" \
PREFIX="$tmp_dir/prefix" \
CONFIG_DIR="$tmp_dir/config" \
STATE_DIR="$tmp_dir/state" \
LOG_DIR="$tmp_dir/log" \
START_SERVICE=false \
  sh "$repo_root/packaging/linux/install.sh" "$binary_source" >/dev/null

if ! grep -Fq '"addr": "0.0.0.0:35338"' "$tmp_dir/config/fleetpulse.json"; then
  echo "installer did not migrate addr to 0.0.0.0:35338" >&2
  cat "$tmp_dir/config/fleetpulse.json" >&2
  exit 1
fi

if ! grep -Fq '"auth_enabled": true' "$tmp_dir/config/fleetpulse.json"; then
  echo "installer did not migrate auth_enabled to true" >&2
  cat "$tmp_dir/config/fleetpulse.json" >&2
  exit 1
fi

if ! grep -Fq '"token_file": "/custom/token"' "$tmp_dir/config/fleetpulse.json"; then
  echo "installer rewrote unrelated config fields" >&2
  cat "$tmp_dir/config/fleetpulse.json" >&2
  exit 1
fi

if [ ! -f "$tmp_dir/systemd/fleetpulse.service" ]; then
  echo "installer did not install systemd service" >&2
  exit 1
fi
