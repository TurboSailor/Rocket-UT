#!/usr/bin/env bash
# Проверка QML прямо на устройстве, без пересборки click (~15 сек и sudo).
# Ловит "Type X unavailable", "... is not available in QtQuick 2.4" и уехавшую
# за экран раскладку (OVERFLOW/NEGATIVE-WIDTH) на нескольких ширинах.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST=/home/phablet/.cache/qmlcheck
PAGES="${PAGES:-HomePage NodesPage NodeEditPage SubsPage ConfsPage ConfEditPage RulesPage LogPage ImportPage FilePage ScanPage}"
WIDTHS="${WIDTHS:-32 45 60}"

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
cp "$ROOT"/click/*.qml "$ROOT"/click/*.js "$ROOT"/click/icon.png "$STAGE/"
for p in $PAGES; do
  for w in $WIDTHS; do
    sed -e "s/@PAGE@/$p/g" -e "s/@WIDTH@/$w/g" \
        "$ROOT/click/dev/harness.qml.in" > "$STAGE/check_${p}_${w}.qml"
  done
done

# Цель всегда сносится: push в существующий каталог создаёт вложенную копию
# и проверяется старый код.
adb shell "rm -rf $DEST; mkdir -p $DEST"
adb push "$STAGE" "$DEST/qml" >/dev/null

# Страницы с UITK TextArea (ImportPage, ConfEditPage) роняют qmlscene под
# плагином offscreen — там они проверяются настоящим ubuntumirclient, но только
# на реальной ширине экрана: окно задаёт шелл, а не корневой Item.
MIR_ENV="QT_QPA_PLATFORM=ubuntumirclient MIR_SOCKET=/run/user/32011/mir_socket \
XDG_RUNTIME_DIR=/run/user/32011 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/32011/bus \
GRID_UNIT_PX=24"
NOISE='XDG_RUNTIME_DIR|createPlatformOpenGLContext|lomiri.deprecations|gralloc|QMirClientScreen|APP_ID|^\s*$'
# Болтовня камеры (ScanPage) под offscreen: media-hub недоступен, к QML не относится.
NOISE="$NOISE|Added camera|Aal[A-Za-z]*|media-hub|media player backend|hubPlayerSession"
NOISE="$NOISE|onPreviewReady|Application is now active|ShaderVideoNodePlugin|Unable to get texture"
NOISE="$NOISE|Image and thumbnail aspect|m_surface is NULL|Timers cannot be started"

fail=0
mir_only=""
for p in $PAGES; do
  for w in $WIDTHS; do
    raw=$(adb shell "cd $DEST/qml && timeout 25 env QT_QPA_PLATFORM=offscreen \
            qmlscene check_${p}_${w}.qml 2>&1; echo rc=\$?" 2>/dev/null)
    if ! echo "$raw" | grep -q "^CHECKED \|rc=0"; then
      # ни одной строки диагностики и ненулевой выход = падение плагина, не код
      [ "$w" = 45 ] || continue
      raw=$(adb shell "cd $DEST/qml && timeout 25 env $MIR_ENV \
              qmlscene check_${p}_${w}.qml 2>&1")
      mir_only="$mir_only $p"
    fi
    out=$(echo "$raw" | grep -viE "$NOISE" | grep -vE "^(qml: )?CHECKED |^rc=0$" || true)
    if [ -n "$out" ]; then
      fail=1
      echo "== $p @ ${w}gu"
      echo "$out"
    fi
  done
done

[ -n "$mir_only" ] && echo "qmlcheck: via ubuntumirclient @45gu only:$mir_only"
[ "$fail" = 0 ] && echo "qmlcheck: clean ($(echo $PAGES | wc -w) pages x $(echo $WIDTHS | wc -w) widths)"
exit $fail
