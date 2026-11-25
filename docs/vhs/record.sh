#!/usr/bin/env bash
# Record the README GIFs from the VHS tapes in this directory.
#
# Usage:
#   ./docs/vhs/record.sh                 # record every tape
#   ./docs/vhs/record.sh auto-switch     # record one tape by name
#
# Requires vhs (with ttyd and ffmpeg), starship, git, go, and ssh-keygen.

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$repo_root"

if ! command -v vhs >/dev/null 2>&1; then
  echo "vhs is not installed; see https://github.com/charmbracelet/vhs" >&2
  exit 1
fi

mkdir -p docs/gifs

tapes=("$@")
if [ ${#tapes[@]} -eq 0 ]; then
  for path in docs/vhs/*.tape; do
    [ -e "$path" ] || continue
    name=${path##*/}
    tapes+=("${name%.tape}")
  done
fi

for tape in "${tapes[@]}"; do
  name=${tape%.tape}
  if [ ! -f "docs/vhs/$name.tape" ]; then
    echo "no tape named $name in docs/vhs" >&2
    exit 1
  fi
  echo "recording $name"
  vhs "docs/vhs/$name.tape"
done
