#!/bin/sh
# Uninstall the Hive (your folder, ~/hive by default, is left alone):
#   curl -fsSL https://raw.githubusercontent.com/ihsan-sa/hive-standalone-build/main/install/uninstall.sh | sh
set -eu
DIR="${HIVE_BIN_DIR:-$HOME/.local/bin}"
if [ -x "$DIR/hive" ]; then "$DIR/hive" uninstall || true; rm -f "$DIR/hive" "$DIR/hive.old"; fi
echo "The Hive is uninstalled. Your folder and its projects are untouched."
