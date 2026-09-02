#!/usr/bin/env bash
# Деплой на устройство по adb.
#
# Пароль sudo НЕ хранится в репозитории. Есть два режима:
#   UT_PASS=...  make deploy   — неинтерактивно (пароль попадёт в историю оболочки
#                                и в список процессов на устройстве)
#   make deploy                — интерактивно: install.sh спросит пароль сам
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VER="${VER:-0.2.1}"
DEST=/home/phablet/Downloads
CLICK="$ROOT/rocket_${VER}_all.click"

[ -f "$CLICK" ] || { echo "missing $CLICK — run: make click"; exit 1; }
[ -x "$ROOT/vendor-bin/rocketd" ] || { echo "missing rocketd — run: make daemon"; exit 1; }

adb devices | sed -n '2p'
echo "== push"
adb push "$CLICK" "$DEST/" >/dev/null
adb push "$ROOT/vendor-bin/rocketd" "$DEST/rocketd" >/dev/null
adb push "$ROOT/scripts/install.sh" "$DEST/install.sh" >/dev/null

echo "== install"
if [ -n "${UT_PASS:-}" ]; then
  # stdin для sudo -S: пароль не появляется в argv на устройстве.
  printf '%s\n' "$UT_PASS" |
    adb shell "sudo -S -p '' env VER=$VER bash $DEST/install.sh 2>&1 | tail -30"
else
  echo "UT_PASS is not set — run the installer on the device and enter the password:"
  echo "  adb shell"
  echo "  sudo env VER=$VER bash $DEST/install.sh"
  exit 0
fi
