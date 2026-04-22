---
title: Diagnostic Discipline
audience: LT, Top, Sarge, anyone who might raise an alarm about system state
last-updated: 2026-04-22
author: Scribe
---

# Diagnostic Discipline

Your default instinct on anomalous state is wrong. Every time an LT session has sounded an alarm from first-glance state, Doc has found the real answer by calmly verifying authoritative sources. This doc encodes the four rules that prevent that failure mode from repeating.

Diagnostic Discipline is a specific instance of the [Self-Correction Loop](./SELF-CORRECTION.md) — it is the rulebook that grew out of repeatedly being wrong about the same class of thing.

## The 4 rules

### Rule 1: Verify origin before declaring work "lost"

When local state looks broken — empty merge queue, stale clone, missing commits, failed-to-ship signals — **check `origin/<default-branch>` first.** Camp Leatherneck has a documented workflow where LT hand-cherry-picks polecat branches, pushes them to origin, and closes beads manually. That leaves local signals (MQ empty, clone stale, bead status odd) that look like failure but aren't. The work is usually on origin via a path you didn't expect.

**Before alarming:**

```bash
cd <rig>/refinery/rig && git fetch origin && git log origin/main --oneline | head -20
# Grep for the bead IDs / commit titles you think are missing
```

Only if `origin/main` truly doesn't have the work should you raise the alarm.

**Example:** `2026-04-22` — MQ on `jcmd_website` looked silent. Real state: LT had cherry-picked and pushed directly. Work was on `origin/main` the whole time.

### Rule 2: Check polecat *state*, not *activity*

`gt polecat list` shows tmux-alive as "working." It is misleading. A finished polecat sitting idle at a prompt emits zero tmux output and looks identical to a hung one. Activity is not progress.

**Before declaring a polecat stuck:**

```bash
gt polecat status <rig>/<name>        # read the actual State: field
tmux capture-pane -t <session> -p -S -100 | tail -50   # read what it last said
git -C <worktree> log main..HEAD      # check for completion commits
```

Look for `gt done`, `Work submitted`, `Polecat is now idle` — those are **completion** signals, not stall signals. If all three sources point to "stuck mid-operation," then alarm.

**Example:** `2026-04-22` — Four `alto_platform` polecats "stalled." Real state: all four had finished cleanly via `gt done` and were idle at a prompt. Activity clock conflated no-stdout with no-progress.

### Rule 3: Build a concrete hypothesis before a worst-case narrative

When you see anomalous state, write down four things — in order — before you speak:

1. **What I observed.** Facts only, no interpretation.
2. **What the authoritative source says.** Origin, `gt polecat status`, bead DB, actual tmux buffer — not your cached mental model.
3. **Competing hypotheses.** At minimum two: a benign explanation and a real problem. Consider benign first.
4. **Cheapest test that discriminates.** Pick one command that would prove or disprove the worst case.

Run the test. Then speak.

**Example:** `2026-04-22` — "The Gas Town repo looks private" was the narrative. Facts said unauthenticated `curl` got a redirect. Benign hypothesis: GitHub redirects anonymous browsers even on public repos. Discriminating test: `gh repo view steveyegge/gastown --json visibility,licenseInfo`. Result: public, MIT. Narrative was wrong; the benign hypothesis was right.

### Rule 4: Doc first, alarm second

If you are about to escalate, mail the Overseer, or broadcast *"something is broken"* — first ask: **would Doc agree after 5 tool calls of verification?**

If you cannot confidently answer yes, dispatch Doc (or run the verification yourself) **before** alarming. The cost of a false alarm is high: Joey's trust compounds, and two false alarms in one conversation damages it measurably. A verified alarm is free; an unverified one spends trust.

## Running list of false-alarm patterns caught

This list grows over time. Each entry is a pattern caught in the wild — a reminder of a specific failure mode so the next session recognizes it earlier.

- **2026-04-22** — MQ silent-failure alarm on `jcmd_website`. Actual cause: LT cherry-pick bypass; work was on `origin/main`. Caught by Doc triage (bead `hq-v9r`). **Rule invoked: #1.**
- **2026-04-22** — Four `alto_platform` polecats flagged "stalled." Actual state: all four had finished cleanly via `gt done` and were idle. tmux activity clock conflated "no stdout" with "no progress." Caught by Doc triage. **Rule invoked: #2.**
- **2026-04-22** — "Gastown repo is private" alarm. Actual cause: unauthenticated `curl` got a redirect that GitHub serves to anonymous browsers, making the repo look private. Repo is public + MIT. **Rule: use `gh repo view <owner>/<repo> --json visibility,licenseInfo` for GitHub state queries, not `curl`.** Rule invoked: #3.

Add to this list every time a false alarm gets caught. The list is seed doctrine, not history — each entry is load-bearing for the next session's hesitation.

## See also

- [`SELF-CORRECTION.md`](./SELF-CORRECTION.md) — the meta-loop that produced these rules
- [`ITERATIVE-LEARNING.md`](./ITERATIVE-LEARNING.md) — how the running list propagates to future sessions
- [`directives/mayor.md`](../directives/mayor.md) — canonical source inside LT's directive
- [`directives/doc.md`](../directives/doc.md) — the specialist most often invoked by Rule 4
