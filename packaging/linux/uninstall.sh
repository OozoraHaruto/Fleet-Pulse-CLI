#!/usr/bin/env sh
set -eu

PREFIX="${PREFIX:-/usr/local}"
CONFIG_DIR="${CONFIG_DIR:-/etc/fleetpulse}"
STATE_DIR="${STATE_DIR:-/var/lib/fleetpulse}"
LOG_DIR="${LOG_DIR:-/var/log/fleetpulse}"
PURGE="${PURGE:-false}"

if command -v systemctl >/dev/null 2>&1; then
  systemctl stop fleetpulse.service >/dev/null 2>&1 || true
  systemctl disable fleetpulse.service >/dev/null 2>&1 || true
  rm -f /etc/systemd/system/fleetpulse.service
  systemctl daemon-reload >/dev/null 2>&1 || true
fi

rm -f "$PREFIX/bin/fleetpulse"

if [ "$PURGE" = "true" ]; then
  rm -rf "$CONFIG_DIR" "$STATE_DIR" "$LOG_DIR"
  echo "FleetPulse uninstalled and state purged."
else
  echo "FleetPulse uninstalled. Preserved $CONFIG_DIR and $STATE_DIR."
fi
