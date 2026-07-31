#!/usr/bin/env bash
# rebuild-gt/run.sh — Rebuild the lt binary from source if stale.
#
# SAFETY: Only rebuilds forward (binary is ancestor of HEAD) and only
# from main branch. A bad rebuild caused a crash loop (every session's
# startup hook failed, witness respawned, loop repeated every 1-2 min).
#
# Fixed 2026-07-31 (LT_IMPLEMENTATION_CONTRACT.md Phase 3 item 8): this
# plugin pointed at ${TOWN_ROOT}/gastown/mayor/rig, a path that has never
# existed (gastown is not a registered rig — see rigs.json), so it silently
# no-op'd on every dispatch. The canonical source root is a single fixed
# location (~/camp-leatherneck, per the Architecture Constitution), not
# town-relative, so there's no TOWN_ROOT to resolve here anymore.

set -euo pipefail

SOURCE_ROOT="${HOME}/camp-leatherneck"

log() { echo "[rebuild-gt] $*"; }

# --- Detection ---------------------------------------------------------------

log "Checking binary staleness..."
STALE_JSON=$(lt stale --json 2>/dev/null) || {
  log "lt stale --json failed, skipping"
  exit 0
}

IS_STALE=$(echo "$STALE_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('stale', False))" 2>/dev/null || echo "False")
SAFE=$(echo "$STALE_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('safe_to_rebuild', False))" 2>/dev/null || echo "False")

if [ "$IS_STALE" != "True" ]; then
  log "Binary is fresh. Nothing to do."
  bd create "rebuild-gt: binary is fresh" -t chore --ephemeral \
    -l type:plugin-run,plugin:rebuild-gt,rig:gastown,result:success \
    --silent 2>/dev/null || true
  exit 0
fi

if [ "$SAFE" != "True" ]; then
  log "Not safe to rebuild (not on main or would be a downgrade). Skipping."
  bd create "Plugin: rebuild-gt [skipped]" -t chore --ephemeral \
    -l type:plugin-run,plugin:rebuild-gt,rig:gastown,result:skipped \
    -d "Skipped: not safe to rebuild" --silent 2>/dev/null || true
  exit 0
fi

# --- Pre-flight checks -------------------------------------------------------

log "Pre-flight checks..."

if [ ! -d "$SOURCE_ROOT" ]; then
  log "Source root $SOURCE_ROOT does not exist. Skipping."
  exit 0
fi

DIRTY=$(git -C "$SOURCE_ROOT" status --porcelain 2>/dev/null)
if [ -n "$DIRTY" ]; then
  log "Repo is dirty, skipping rebuild."
  bd create "Plugin: rebuild-gt [skipped]" -t chore --ephemeral \
    -l type:plugin-run,plugin:rebuild-gt,rig:gastown,result:skipped \
    -d "Skipped: repo has uncommitted changes" --silent 2>/dev/null || true
  exit 0
fi

BRANCH=$(git -C "$SOURCE_ROOT" branch --show-current 2>/dev/null)
if [ "$BRANCH" != "main" ]; then
  log "Not on main branch (on $BRANCH), skipping rebuild."
  bd create "Plugin: rebuild-gt [skipped]" -t chore --ephemeral \
    -l type:plugin-run,plugin:rebuild-gt,rig:gastown,result:skipped \
    -d "Skipped: not on main branch (on $BRANCH)" --silent 2>/dev/null || true
  exit 0
fi

# --- Build -------------------------------------------------------------------

OLD_VER=$(lt version 2>/dev/null | head -1 || echo "unknown")
log "Rebuilding gt from $SOURCE_ROOT..."

if (cd "$SOURCE_ROOT" && make build && make safe-install) 2>&1; then
  NEW_VER=$(lt version 2>/dev/null | head -1 || echo "unknown")
  log "Rebuilt: $OLD_VER -> $NEW_VER"
  bd create "rebuild-gt: $OLD_VER -> $NEW_VER" -t chore --ephemeral \
    -l type:plugin-run,plugin:rebuild-gt,rig:gastown,result:success \
    --silent 2>/dev/null || true
else
  ERROR="make build/safe-install failed"
  log "FAILED: $ERROR"
  bd create "Plugin: rebuild-gt [failure]" -t chore --ephemeral \
    -l type:plugin-run,plugin:rebuild-gt,rig:gastown,result:failure \
    -d "Build failed: $ERROR" --silent 2>/dev/null || true
  lt escalate "Plugin FAILED: rebuild-gt" -s medium 2>/dev/null || true
  exit 1
fi
