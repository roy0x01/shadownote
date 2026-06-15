#!/usr/bin/env bash
set -euo pipefail
root="${1:-internal/gui/static/css}"
if grep -RInE 'font-weight[[:space:]]*:[[:space:]]*(600|700|800|900|bold|bolder)' "$root" --include='*.css'; then
  echo "Typography audit failed: dashboard CSS must not use bold/heavy weights." >&2
  exit 1
fi
if grep -RInE '@import.*Geist\+Mono.*(600|700|800|900)' "$root" --include='*.css'; then
  echo "Typography audit failed: Geist Mono import should only request 400/500." >&2
  exit 1
fi
echo "Typography audit OK"
