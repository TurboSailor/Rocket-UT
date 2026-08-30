#!/usr/bin/env bash
# Готовит бинарные зависимости в vendor-bin/ (в git не хранятся).
#
#   sing-box       — официальный релиз linux/arm64 (+ darwin для локальных тестов)
#   amneziawg-go   — собирается из исходников (чистый Go, кросс-компиляция)
#   awg            — CLI amneziawg-tools; нужен C-кросс-компилятор ИЛИ AWG_BIN_DIR
#   zxing          — декодер QR; берётся из AWG_BIN_DIR (готовый бинарь)
#
# Переменные окружения:
#   SB_VER       версия sing-box (по умолчанию 1.13.21)
#   AWG_BIN_DIR  каталог с готовыми amneziawg-go-arm64 / awg-arm64 / zxing-arm64
#   AWG_CLICK    путь к click-пакету awg-control, из которого их можно извлечь
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/vendor-bin"
mkdir -p "$OUT"
SUMFILE="$OUT/SHA256SUMS"

SB_VER="${SB_VER:-1.13.21}"
AWG_BIN_DIR="${AWG_BIN_DIR:-}"
AWG_CLICK="${AWG_CLICK:-}"

sum() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'
  else shasum -a 256 "$1" | awk '{print $1}'; fi
}

fetch_singbox() {
  local goos="$1" arch="$2" dest="$3"
  [ -x "$dest" ] && { echo "have $(basename "$dest")"; return 0; }
  local tmpd; tmpd="$(mktemp -d)"
  echo "GET sing-box ${SB_VER} ${goos}-${arch}"
  curl -fL --retry 3 -o "$tmpd/sb.tgz" \
    "https://github.com/SagerNet/sing-box/releases/download/v${SB_VER}/sing-box-${SB_VER}-${goos}-${arch}.tar.gz"
  tar -xzf "$tmpd/sb.tgz" -C "$tmpd"
  install -m 755 "$(find "$tmpd" -type f -name sing-box | head -1)" "$dest"
  rm -rf "$tmpd"
}

fetch_singbox linux arm64 "$OUT/sing-box"
# Хостовый бинарь используется тестами для валидации конфига (`sing-box check`).
case "$(uname -s)" in
  Darwin) fetch_singbox darwin arm64 "$OUT/sing-box-darwin" ;;
esac

# --- amneziawg-go: чистый Go, собирается кросс-компиляцией ---
if [ ! -x "$OUT/amneziawg-go" ]; then
  if [ -n "$AWG_BIN_DIR" ] && [ -x "$AWG_BIN_DIR/amneziawg-go-arm64" ]; then
    install -m 755 "$AWG_BIN_DIR/amneziawg-go-arm64" "$OUT/amneziawg-go"
  else
    tmpd="$(mktemp -d)"
    echo "BUILD amneziawg-go (linux/arm64)"
    git clone --depth 1 https://github.com/amnezia-vpn/amneziawg-go "$tmpd/src"
    (cd "$tmpd/src" && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
      go build -trimpath -ldflags "-s -w" -o "$OUT/amneziawg-go" .)
    rm -rf "$tmpd"
  fi
fi

# --- awg и zxing: готовые бинари (сборка awg требует C-кросс-компилятора) ---
need_extract=0
[ -x "$OUT/awg" ] || need_extract=1
[ -x "$OUT/zxing" ] || need_extract=1
if [ "$need_extract" = 1 ]; then
  src="$AWG_BIN_DIR"
  if [ -z "$src" ] && [ -n "$AWG_CLICK" ] && [ -f "$AWG_CLICK" ]; then
    tmpd="$(mktemp -d)"
    (cd "$tmpd" && ar x "$AWG_CLICK" && tar -xzf data.tar.gz)
    src="$tmpd/bin"
  fi
  if [ -n "$src" ] && [ -x "$src/awg-arm64" ]; then
    install -m 755 "$src/awg-arm64" "$OUT/awg"
    [ -x "$src/zxing-arm64" ] && install -m 755 "$src/zxing-arm64" "$OUT/zxing"
  else
    cat >&2 <<'EOF'
ERROR: awg/zxing not found.
Provide prebuilt aarch64 binaries via AWG_BIN_DIR (expects awg-arm64, zxing-arm64),
or point AWG_CLICK at an awg-control .click that contains them.
Building awg from source needs a C cross-compiler:
  apt install gcc-aarch64-linux-gnu libc6-dev-arm64-cross
  git clone --depth 1 https://github.com/amnezia-vpn/amneziawg-tools
  make -C amneziawg-tools/src CC=aarch64-linux-gnu-gcc
  # результат: amneziawg-tools/src/wg -> vendor-bin/awg
EOF
    exit 1
  fi
fi

{
  for f in sing-box sing-box-darwin amneziawg-go awg zxing; do
    [ -f "$OUT/$f" ] && echo "$(sum "$OUT/$f")  $f"
  done
} > "$SUMFILE"
echo "wrote $SUMFILE"
cat "$SUMFILE"
