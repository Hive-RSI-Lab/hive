#!/bin/sh
# Uninstall the Hive (your folder, ~/hive by default, is left alone):
#   gh api repos/ihsan-sa/hive-standalone-build/contents/install/uninstall.sh -H "Accept: application/vnd.github.raw" | sh
set -eu
CMD="${HIVE_CMD:-hive-v0}"
DIR="${HIVE_BIN_DIR:-$HOME/.local/bin}"
if [ -x "$DIR/$CMD" ]; then "$DIR/$CMD" uninstall || true; rm -f "$DIR/$CMD"; fi
echo "The Hive is uninstalled. Your folder and its projects are untouched."
