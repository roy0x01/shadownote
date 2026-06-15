#!/usr/bin/env sh
set -eu
root="${1:-internal/gui/static/css}"
fail=0

if [ -e "$root/app.css" ]; then
  echo "ERROR: app.css must not exist" >&2
  fail=1
fi

# Page files must not own global shell/nav selectors.
for f in "$root"/pages/*.css; do
  [ "$(basename "$f")" = "editor.css" ] && continue
  if grep -Eq '(^|[,{[:space:]])\.(app-shell|workspace|side|brand|side-nav|nav-item|page-header|page-shell|page)([[:space:].:{,#]|$)' "$f"; then
    echo "ERROR: page css owns global/nav selector: $f" >&2
    fail=1
  fi
done

# Core must not contain known page-specific roots.
if grep -Eq '\.(dashboard|content-shell|assets-page|asset-|editor|ed-|setting-panel|doc-row|doc-list|stats)' "$root/core/main.css"; then
  echo "ERROR: core/main.css contains page/component-specific selectors" >&2
  fail=1
fi

# Nav component should be the only CSS file with sidebar selectors.
for f in "$root"/core/*.css "$root"/components/*.css "$root"/pages/*.css; do
  base="$(basename "$f")"
  [ "$base" = "nav.css" ] && continue
  [ "$base" = "editor.css" ] && continue
  if grep -Eq '\.(side|brand|side-nav|nav-item|nav-newdraft)' "$f"; then
    echo "ERROR: non-nav css owns sidebar selector: $f" >&2
    fail=1
  fi
done

exit "$fail"
