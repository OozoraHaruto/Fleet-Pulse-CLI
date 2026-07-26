#!/usr/bin/env sh
set -eu

PREFIX="${PREFIX:-/usr/local}"
CONFIG_DIR="${CONFIG_DIR:-/Library/Application Support/FleetPulse}"
PLIST="/Library/LaunchDaemons/com.fleetpulse.agent.plist"
PURGE="${PURGE:-false}"

launchctl bootout system "$PLIST" >/dev/null 2>&1 || true
rm -f "$PLIST"
rm -f "$PREFIX/bin/fleetpulse"

if [ "$PURGE" = "true" ]; then
  rm -rf "$CONFIG_DIR"
  echo "FleetPulse uninstalled and state purged."
else
  echo "FleetPulse uninstalled. Preserved $CONFIG_DIR."
fi
