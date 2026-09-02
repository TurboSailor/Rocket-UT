#!/usr/bin/env bash
# Сборка click-пакета на macOS: click(1) здесь нет, поэтому ar+tar вручную.
# Раскладка воспроизводит рабочий awg-control_0.2.1_all.click.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VER="${VER:-0.2.0}"
SRC="$ROOT/click"
BIN="$ROOT/vendor-bin"
OUT="$ROOT/rocket_${VER}_all.click"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

for f in sing-box amneziawg-go awg zxing; do
  [ -x "$BIN/$f" ] || { echo "missing $BIN/$f — run: make bins"; exit 1; }
done

# --- data ---
D="$WORK/data"
mkdir -p "$D/bin"
cp "$SRC"/*.qml "$SRC"/*.js "$SRC/rocket.desktop" "$SRC/rocket.apparmor" "$D/"
cp "$SRC/icon.png" "$D/icon.png"
# Штатный шаблон правил: демон сидирует его на чистой установке.
cp "$SRC/default.conf" "$D/default.conf"
install -m 755 "$BIN/sing-box" "$D/bin/sing-box"
install -m 755 "$BIN/amneziawg-go" "$D/bin/amneziawg-go"
install -m 755 "$BIN/awg" "$D/bin/awg"
install -m 755 "$BIN/zxing" "$D/bin/zxing"
chmod 644 "$D"/*.qml "$D"/*.js "$D"/rocket.desktop "$D"/rocket.apparmor "$D"/icon.png "$D"/default.conf

INSTALLED_KB=$(du -sk "$D" | awk '{print $1}')

# --- control ---
C="$WORK/control"
mkdir -p "$C"
sed "s/\"version\": \"[^\"]*\"/\"version\": \"${VER}\"/" "$SRC/manifest.json" > "$C/manifest"
python3 - "$C/manifest" "$INSTALLED_KB" <<'PY'
import json, sys
p, size = sys.argv[1], sys.argv[2]
m = json.load(open(p))
m["installed-size"] = str(size)
json.dump(m, open(p, "w"), indent=4, sort_keys=True)
PY

cat > "$C/control" <<EOF
Package: rocket
Version: ${VER}
Click-Version: 0.4
Architecture: all
Maintainer: local <local@localhost>
Installed-Size: ${INSTALLED_KB}
Description: Rocket proxy client
EOF

cat > "$C/preinst" <<'EOF'
#! /bin/sh
echo "Click packages may not be installed directly using dpkg."
echo "Use 'click install' instead."
exit 1
EOF
chmod 755 "$C/preinst"

( cd "$D" && find . -type f -print0 | sort -z | xargs -0 md5sum | sed 's|\./||' ) > "$C/md5sums"

# --- assemble (порядок членов ar обязателен) ---
echo "2.0" > "$WORK/debian-binary"
echo "0.4" > "$WORK/_click-binary"
# COPYFILE_DISABLE=1 — иначе bsdtar кладёт AppleDouble-записи ._* и click отказывает.
( cd "$C" && COPYFILE_DISABLE=1 tar --no-xattrs --no-mac-metadata -czf "$WORK/control.tar.gz" . )
( cd "$D" && COPYFILE_DISABLE=1 tar --no-xattrs --no-mac-metadata -czf "$WORK/data.tar.gz" . )

rm -f "$OUT"
# BSD ar на macOS вставляет __.SYMDEF и теряет члены — пишем ar-архив сами.
python3 - "$WORK" "$OUT" <<'PY'
import sys, os
work, out = sys.argv[1], sys.argv[2]
members = ["debian-binary", "_click-binary", "control.tar.gz", "data.tar.gz"]
with open(out, "wb") as f:
    f.write(b"!<arch>\n")
    for name in members:
        p = os.path.join(work, name)
        data = open(p, "rb").read()
        hdr = (name.ljust(16) + "0".ljust(12) + "0".ljust(6) + "0".ljust(6)
               + "100644".ljust(8) + str(len(data)).ljust(10) + "`\n")
        f.write(hdr.encode())
        f.write(data)
        if len(data) % 2:
            f.write(b"\n")
PY

echo "built $OUT ($(du -h "$OUT" | awk '{print $1}'))"
ar t "$OUT"
