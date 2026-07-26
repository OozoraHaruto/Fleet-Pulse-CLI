#!/usr/bin/env sh
set -eu

PREFIX="${PREFIX:-/usr/local}"
CONFIG_DIR="${CONFIG_DIR:-/Library/Application Support/FleetPulse}"
STATE_DIR="${STATE_DIR:-/Library/Application Support/FleetPulse/state}"
PLIST="/Library/LaunchDaemons/com.fleetpulse.agent.plist"
START_SERVICE="${START_SERVICE:-true}"
BINARY_SOURCE="${1:-./fleetpulse}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd -P)"
PLIST_SOURCE="$SCRIPT_DIR/com.fleetpulse.agent.plist"

clear_quarantine() {
  path="$1"
  if command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$path" >/dev/null 2>&1 || true
  fi
}

if [ ! -f "$BINARY_SOURCE" ]; then
  echo "fleetpulse binary not found at $BINARY_SOURCE" >&2
  exit 1
fi

if [ ! -f "$PLIST_SOURCE" ]; then
  echo "FleetPulse launch daemon plist not found at $PLIST_SOURCE" >&2
  exit 1
fi

clear_quarantine "$BINARY_SOURCE"

install -d -m 0755 "$PREFIX/bin"
install -d -m 0750 "$CONFIG_DIR" "$STATE_DIR"
install -m 0755 "$BINARY_SOURCE" "$PREFIX/bin/fleetpulse"
clear_quarantine "$PREFIX/bin/fleetpulse"

if [ ! -f "$CONFIG_DIR/fleetpulse.json" ]; then
  cat >"$CONFIG_DIR/fleetpulse.json" <<EOF
{
  "addr": "127.0.0.1:35338",
  "auth_enabled": false,
  "token_file": "$STATE_DIR/token",
  "cache_ttl": "10s",
  "collector_timeout": "2s",
  "log_level": "info",
  "service_name": "fleetpulse",
  "deployment_target": "macos"
}
EOF
  chmod 0640 "$CONFIG_DIR/fleetpulse.json"
fi

if ! "$PREFIX/bin/fleetpulse" token show -token-file "$STATE_DIR/token" >/dev/null 2>&1; then
  "$PREFIX/bin/fleetpulse" token rotate -token-file "$STATE_DIR/token" >/dev/null
fi
chmod 0600 "$STATE_DIR/token"

install -m 0644 "$PLIST_SOURCE" "$PLIST"
if [ "$START_SERVICE" = "true" ]; then
  launchctl bootout system "$PLIST" >/dev/null 2>&1 || true
  launchctl bootstrap system "$PLIST"
fi

echo "FleetPulse installed. Token is stored at $STATE_DIR/token."
