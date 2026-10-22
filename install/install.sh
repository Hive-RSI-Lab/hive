#!/bin/sh
# Install the Hive on Linux or macOS:
#   curl -fsSL https://raw.githubusercontent.com/ihsan-sa/hive/main/install/install.sh | sh
# Running it again reinstalls. The binary is static; no runtime is needed.
set -eu

REPO="${HIVE_REPO:-ihsan-sa/hive}"
DIR="${HIVE_BIN_DIR:-$HOME/.local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "hive: unsupported architecture $(uname -m)" >&2; exit 1 ;;
esac
case "$os" in linux|darwin) ;; *) echo "hive: unsupported OS $os (on Windows use install.ps1)" >&2; exit 1 ;; esac

asset="hive-$os-$arch"
base="https://github.com/$REPO/releases/latest/download"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading the Hive ($asset)..."
curl -fsSL "$base/$asset" -o "$tmp/hive"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"
want=$(grep " $asset\$" "$tmp/checksums.txt" | awk '{print $1}')
if command -v sha256sum >/dev/null 2>&1; then got=$(sha256sum "$tmp/hive" | awk '{print $1}'); else got=$(shasum -a 256 "$tmp/hive" | awk '{print $1}'); fi
if [ -z "$want" ] || [ "$want" != "$got" ]; then echo "hive: checksum mismatch; not installing" >&2; exit 1; fi

mkdir -p "$DIR"
install -m 755 "$tmp/hive" "$DIR/hive"

case ":$PATH:" in
  *":$DIR:"*) ;;
  *)
    for rc in "$HOME/.profile" "$HOME/.bashrc" "$HOME/.zshrc"; do
      [ -f "$rc" ] || [ "$rc" = "$HOME/.profile" ] || continue
      grep -qs "$DIR" "$rc" || printf '\nexport PATH="%s:$PATH"\n' "$DIR" >> "$rc"
    done
    echo "Added $DIR to your PATH (open a new terminal)."
    ;;
esac
echo "Installed $("$DIR/hive" --version). Run: hive"
