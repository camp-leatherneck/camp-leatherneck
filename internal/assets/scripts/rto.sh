#!/bin/bash
# rto.sh — RTO (Radio Telephone Operator) sitrep generator
# Part of the Camp Leatherneck Marine overlay.
# Regenerates ~/Desktop/sitrep.md from town state.
#
# Scheduled via launchd (~/Library/LaunchAgents/com.campleatherneck.rto.plist)
# or callable directly: bash ~/lt/scripts/rto.sh
#
# RTO persona: see ~/lt/directives/rto.md
# Roster: ~/Desktop/camp_leatherneck_roster.csv

set -u  # not -e — we want the script to always produce a sitrep, even on partial failure

export PATH="/opt/homebrew/bin:/usr/local/bin:$HOME/.local/bin:$HOME/go/bin:$PATH"

SITREP_FILE="$HOME/Desktop/sitrep.md"
TMP_FILE="$(mktemp -t sitrep)"
TS="$(date '+%Y-%m-%d %H:%M')"
NEXT_TS="$(date -v+2M '+%H:%M' 2>/dev/null || date -d '+2 minutes' '+%H:%M' 2>/dev/null || echo "auto")"

# macOS may not have `timeout`; detect and prefer gtimeout if available, otherwise run without a timeout.
if command -v gtimeout >/dev/null 2>&1; then
    TIMEOUT_CMD="gtimeout 10"
elif command -v timeout >/dev/null 2>&1; then
    TIMEOUT_CMD="timeout 10"
else
    TIMEOUT_CMD=""
fi

run_cmd() {
    local cmd="$1"
    local output
    output=$(eval "${TIMEOUT_CMD} ${cmd}" 2>&1) || {
        echo "(command failed: ${cmd})"
        return 1
    }
    echo "$output"
}

# Collect data
RIG_LIST=$(run_cmd "lt rig list")
BD_READY=$(run_cmd "bd ready 2>&1 | head -20")
BD_INPROGRESS=$(run_cmd "bd list --status=in_progress 2>&1 | head -20")
MAIL_INBOX=$(run_cmd "lt mail inbox 2>&1 | head -15")
DOLT_STATUS=$(run_cmd "lt dolt status 2>&1 | head -5")

# Parse counts (grep -c always prints the number, even if 0 — no fallback needed)
UNREAD_MAIL=$(echo "$MAIL_INBOX" | grep -oE '[0-9]+ unread' | head -1)
[[ -z "$UNREAD_MAIL" ]] && UNREAD_MAIL="0 unread"
INPROGRESS_COUNT=$(echo "$BD_INPROGRESS" | grep -cE '^[◇○●]' | tr -d '[:space:]')
READY_COUNT=$(echo "$BD_READY" | grep -cE '^[◇○●]' | tr -d '[:space:]')

# Extract rig health
RIG_GREEN=$(echo "$RIG_LIST" | grep -c '🟢' | tr -d '[:space:]')
RIG_DOWN=$(echo "$RIG_LIST" | grep -c '⚫' | tr -d '[:space:]')

# Dolt health
if echo "$DOLT_STATUS" | grep -q -iE 'ok|running|healthy|up'; then
    DOLT_MARK="ok"
else
    DOLT_MARK="unknown"
fi

# Today's closed beads (best-effort)
TODAY=$(date '+%Y-%m-%d')
CLOSED_TODAY=$(bd list --status=closed 2>/dev/null | grep "$TODAY" | head -10 || echo "")
if [[ -z "$CLOSED_TODAY" ]]; then
    CLOSED_COUNT=0
else
    CLOSED_COUNT=$(echo "$CLOSED_TODAY" | wc -l | tr -d '[:space:]')
fi

# Build sitrep
{
    echo "CAMP LEATHERNECK SITREP — ${TS}"
    echo ""

    # 🔴 NEEDS OVERSEER
    echo "🔴 NEEDS OVERSEER                [${UNREAD_MAIL}]"
    if [[ "$UNREAD_MAIL" == "0 unread" ]]; then
        echo "  (none — inbox clear)"
    else
        echo "$MAIL_INBOX" | grep -E '^\s+[0-9]+\.\s+●' | head -5 | sed 's/^/  • /'
    fi
    echo ""

    # 🟡 IN FLIGHT
    echo "🟡 IN FLIGHT                     [${INPROGRESS_COUNT}]"
    if [[ "$INPROGRESS_COUNT" -eq 0 ]]; then
        echo "  (no active convoys)"
    else
        echo "$BD_INPROGRESS" | grep -E '^[◇○●]' | head -10 | sed 's/^/  • /'
    fi
    echo ""

    # 🟢 SHIPPED TODAY
    echo "🟢 SHIPPED TODAY                 [${CLOSED_COUNT}]"
    if [[ "$CLOSED_COUNT" -eq 0 ]]; then
        echo "  (nothing closed today yet)"
    else
        echo "$CLOSED_TODAY" | head -5 | sed 's/^/  • /'
    fi
    echo ""

    # ⚫ READY QUEUE
    echo "⚫ READY QUEUE                   [${READY_COUNT} ready]"
    if [[ "$READY_COUNT" -eq 0 ]]; then
        echo "  (no beads ready)"
    else
        echo "$BD_READY" | grep -E '^[◇○●]' | head -5 | sed 's/^/  • /'
    fi
    echo ""

    # ⚠️  SYSTEMS
    echo "⚠️  SYSTEMS"
    echo "  • Dolt: ${DOLT_MARK}"
    echo "  • Rigs: ${RIG_GREEN} 🟢 / ${RIG_DOWN} ⚫"
    echo ""

    echo "---"
    echo "Last update: ${TS}  |  Next auto-refresh: ${NEXT_TS}"
    echo "Source: \`~/lt/scripts/rto.sh\` (RTO — see \`~/lt/directives/rto.md\`)"
    echo "Roster: \`~/Desktop/camp_leatherneck_roster.csv\`"
} > "$TMP_FILE"

# Atomic swap — sitrep never appears half-written
mv "$TMP_FILE" "$SITREP_FILE"

exit 0
