#!/bin/bash
# Установка Rocket на Ubuntu Touch 24.04 (aarch64). Запуск: sudo bash install.sh
set -euo pipefail
DIR=$(cd "$(dirname "$0")" && pwd)
[ "$(id -u)" = 0 ] || exec sudo bash "$0" "$@"

VER="${VER:-0.1.1}"
CLICK="$DIR/rocket_${VER}_all.click"
[ -f "$CLICK" ] || { echo "missing $CLICK"; exit 1; }
[ -f "$DIR/rocketd" ] || { echo "missing $DIR/rocketd"; exit 1; }

PHABLET_UID=$(id -u phablet 2>/dev/null || echo 32011)

echo "== / -> rw"
mount -o remount,rw / || true
if ! touch /etc/.wtest 2>/dev/null; then
  echo "ERROR: root filesystem is read-only."
  echo "On UT 24.04: sudo make-writable  (or create /userdata/.writable_image and reboot)"
  exit 1
fi
rm -f /etc/.wtest

echo "== rocketd daemon"
systemctl stop rocketd 2>/dev/null || true
install -m 755 "$DIR/rocketd" /usr/local/bin/rocketd
mkdir -p /var/lib/rocketd/confs /var/lib/rocketd/awg /var/lib/rocketd/rulesets /var/lib/rocketd/bin
mkdir -p /var/run/amneziawg
chmod 700 /var/lib/rocketd /var/lib/rocketd/confs /var/lib/rocketd/awg
chmod 700 /var/run/amneziawg

cat > /etc/systemd/system/rocketd.service <<'EOF'
[Unit]
Description=Rocket proxy root daemon (HTTP 127.0.0.1:8877)
After=network.target

[Service]
ExecStart=/usr/local/bin/rocketd
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable rocketd
systemctl restart rocketd

echo "== click ${VER}"
export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-unix:path=/run/user/${PHABLET_UID}/bus}"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/${PHABLET_UID}}"

# Хук desktop не перезаписывает уже сгенерированную запись, если имя файла
# не изменилось, поэтому при переустановке той же версии лончер продолжает
# показывать старую иконку и старый splash. Сносим устаревшие записи сами.
PH_HOME=$(getent passwd phablet | cut -d: -f6)
PH_HOME="${PH_HOME:-/home/phablet}"
rm -f "$PH_HOME"/.local/share/applications/rocket_rocket_*.desktop
rm -f "$PH_HOME"/.cache/lomiri-app-launch/desktop/rocket_rocket_*.desktop
# Известная грабля UT: полу-установка оставляет каталог и install падает в hooks.
if ! click install --force --allow-unauthenticated --user=phablet "$CLICK" 2>&1 | tail -5; then
  echo "-- retry after cleaning /opt/click.ubuntu.com/rocket"
  rm -rf /opt/click.ubuntu.com/rocket
  click install --force --allow-unauthenticated --user=phablet "$CLICK" 2>&1 | tail -5
fi
click register --user=phablet rocket "${VER}" 2>/dev/null || true
aa-clickhook -f 2>/dev/null || true

# Бинари доступны демону и из click, но локальная копия спасает при пересборке UI.
CLICK_BIN=/opt/click.ubuntu.com/rocket/current/bin
if [ -d "$CLICK_BIN" ]; then
  for f in sing-box amneziawg-go awg; do
    [ -f "$CLICK_BIN/$f" ] && install -m 755 "$CLICK_BIN/$f" "/var/lib/rocketd/bin/$f"
  done
fi
systemctl restart rocketd

sleep 2
echo ""
echo "== rocketd: $(systemctl is-active rocketd)"
wget -q -O- http://127.0.0.1:8877/status && echo ""
echo ""
echo "== / -> ro"
mount -o remount,ro / || true
echo "Done. Tap the Rocket icon in the launcher (not via adb)."
