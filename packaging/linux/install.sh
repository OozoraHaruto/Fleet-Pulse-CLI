#!/usr/bin/env sh
set -eu

PREFIX="${PREFIX:-/usr/local}"
CONFIG_DIR="${CONFIG_DIR:-/etc/fleetpulse}"
STATE_DIR="${STATE_DIR:-/var/lib/fleetpulse}"
LOG_DIR="${LOG_DIR:-/var/log/fleetpulse}"
SERVICE_USER="${SERVICE_USER:-fleetpulse}"
SERVICE_GROUP="${SERVICE_GROUP:-fleetpulse}"
START_SERVICE="${START_SERVICE:-true}"
BINARY_SOURCE="${1:-./fleetpulse}"

if [ ! -f "$BINARY_SOURCE" ]; then
  echo "fleetpulse binary not found at $BINARY_SOURCE" >&2
  exit 1
fi

if ! id "$SERVICE_USER" >/dev/null 2>&1; then
  useradd --system --home "$STATE_DIR" --shell /usr/sbin/nologin "$SERVICE_USER"
fi

install -d -m 0755 "$PREFIX/bin"
install -d -m 0750 -o "$SERVICE_USER" -g "$SERVICE_GROUP" "$CONFIG_DIR" "$STATE_DIR" "$LOG_DIR"
install -m 0755 "$BINARY_SOURCE" "$PREFIX/bin/fleetpulse"

if [ ! -f "$CONFIG_DIR/fleetpulse.json" ]; then
  cat >"$CONFIG_DIR/fleetpulse.json" <<EOF
{
  "addr": "127.0.0.1:8080",
  "auth_enabled": false,
  "token_file": "$STATE_DIR/token",
  "cache_ttl": "10s",
  "collector_timeout": "2s",
  "log_level": "info",
  "service_name": "fleetpulse",
  "deployment_target": "linux"
}
EOF
  chmod 0640 "$CONFIG_DIR/fleetpulse.json"
  chown "$SERVICE_USER:$SERVICE_GROUP" "$CONFIG_DIR/fleetpulse.json"
fi

if ! "$PREFIX/bin/fleetpulse" token show -token-file "$STATE_DIR/token" >/dev/null 2>&1; then
  "$PREFIX/bin/fleetpulse" token rotate -token-file "$STATE_DIR/token" >/dev/null
fi
chown "$SERVICE_USER:$SERVICE_GROUP" "$STATE_DIR/token"
chmod 0600 "$STATE_DIR/token"

install -m 0644 packaging/linux/fleetpulse.service /etc/systemd/system/fleetpulse.service
systemctl daemon-reload
systemctl enable fleetpulse.service
if [ "$START_SERVICE" = "true" ]; then
  systemctl restart fleetpulse.service
fi

echo "FleetPulse installed. Token is stored at $STATE_DIR/token."
echo "Run 'fleetpulse token show -token-file $STATE_DIR/token' as an administrator to retrieve it."
