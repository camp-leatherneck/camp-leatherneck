# Camp Leatherneck

**Run a persistent team of AI coding agents on your own repos.**
`lt` gives you a planner, workers, a merge queue, and a watchdog — so you can hand off a bug, walk away, and come back to a merged PR.

Built on top of [Gas Town](https://github.com/steveyegge/gastown) by Steve Yegge. MIT-licensed. Free.

---

## The problem

One AI coding agent is fine. Running three or five in parallel on the same repo is chaos: they step on each other's branches, one crashes silently at 2am, another starts going in circles on a bad idea, and you end up babysitting them instead of doing real work.

## What Camp Leatherneck does

It gives you a **standing crew** of agents with roles:

- **LT** plans and routes work to the right agent
- **Polecats** do the actual coding, each in an isolated branch
- **Gunny** gates merges through a queue so two agents never clobber each other
- **Top** watches for stalls and restarts stuck agents
- **Sarge** and **Fire Watch** keep the whole thing running unattended

You file work as a **bead** (an issue with structured metadata), hand it to a polecat, and the system coordinates the rest. When an agent ships, Gunny reviews, queues, and merges. If an agent wanders, LT re-plans. If LT misbehaves, a **Self-Correction Loop** forces it to write the rule it violated into its own durable instructions.

## What it looks like

```bash
# You have a bug. File it and sling it to an agent.
$ lt assign rictus "Fix auth token refresh — silently fails after 60min"

# Rictus picks it up, opens a branch, investigates, writes a fix + test.
# Gunny reviews, queues, merges. You get a notification, not a prompt.

# Meanwhile you sling three more small bugs to three other agents.
# They work in parallel. You check back in an hour.

$ lt status
4 polecats active · 2 merged · 1 in review · 1 blocked on you
```

The whole point is trust: agents run **unattended** long enough to finish work, with enough structure (merge queue, role boundaries, diagnostic doctrine) that they don't wreck the repo while you're not watching.

## Who this is for

You, if:

- You already use [Claude Code](https://claude.ai/claude-code) (or Codex / OpenCode / Copilot CLI) and wish you could run more than one at a time
- You have a repo with enough small-to-medium issues that babysitting one agent at a time is the bottleneck
- You're OK on macOS or Linux with a few extra CLI dependencies

Not for you (yet), if:

- You want zero-config SaaS — Camp Leatherneck runs on your machine with your API key
- You want Windows-native — not supported today
- You want no setup — there are prerequisites (see Install)

## What makes it different

Most "AI agent framework" projects give you a loop and a tool list. Camp Leatherneck is **opinionated about how agents should behave when you're not watching**, and that shows up as three real features:

**Self-Correction Loop.** When the planner makes a mistake (bad task decomposition, wrong agent assignment, ignored instruction), it doesn't just apologize and retry — it writes the violated rule into its own persistent instructions so the same mistake can't recur. Your agent team gets stricter every week you run it. See [`docs/SELF-CORRECTION.md`](./docs/SELF-CORRECTION.md).

**Diagnostic Discipline.** Four rules that stop agents from spiraling on false alarms: anomaly → verify → hypothesize → test → *then* escalate. No more 3am pages because one agent saw an unfamiliar directory and assumed the repo was corrupted. See [`docs/DIAGNOSTIC-DISCIPLINE.md`](./docs/DIAGNOSTIC-DISCIPLINE.md).

**Merge queue + role separation.** Borrowed from Gas Town: writers don't merge their own code, and the merge queue (Gunny) serializes so two agents can't both land conflicting changes. This is the feature that makes parallel agents actually safe on the same repo.

## Install

**Prerequisites** (one-time):

- macOS or Linux
- [Claude Code](https://claude.ai/claude-code) ≥ 2.0.20 (or another supported agent provider)
- [Dolt](https://github.com/dolthub/dolt?tab=readme-ov-file#installation) ≥ 1.82.4 — runs the bead/mail data plane
- [Beads](https://github.com/steveyegge/beads) (`go install github.com/steveyegge/beads/cmd/bd@latest`)
- An Anthropic API key (or equivalent for your provider)

**Install `lt`:**

```bash
brew install camp-leatherneck/tap/lt
lt --version   # should print 0.1.0
```

**Create your first HQ:**

```bash
lt install ~/lt           # creates ~/lt with the mayor config, beads DB, etc.
cd ~/lt
lt doctor                 # health-check the install
```

Getting from `lt install` to "my first polecat merged a PR" is ~15 minutes today. A tested cold-machine walkthrough is being written — if you try it and get stuck, file an issue and the friction points become the next release.

## Honest limits (v0.1.0)

- **macOS is the first-class platform.** Linux works but is less exercised.
- **No bundled macOS status app in v0.1.0.** A companion Sitrep.app is planned but waiting on code-signing. The plain-text sitrep at `~/Desktop/sitrep.md` (written every few minutes by the RTO role) is the intended interface in the meantime.
- **API costs are yours.** Each polecat runs on your API key. A morning's work can be $1–$10 depending on how much code moves. Watch `lt costs`.

## Relationship to Gas Town

Camp Leatherneck is a **layered fork** of [Gas Town](https://github.com/steveyegge/gastown). Steve Yegge invented the underlying mechanics: rigs, polecats, beads, convoys, the merge queue, Dolt integration, and the watchdog daemons. Camp Leatherneck renames the user-facing surface to Marine terminology and adds three things on top: the Self-Correction Loop, Diagnostic Discipline, and the RTO sitrep role.

We sync from Gas Town upstream regularly. If a feature ships there that you want here, open an issue — usually it's a one-commit merge.

See [NOTICE](./NOTICE) for attribution and [LICENSE](./LICENSE) for MIT terms.

## Contribute / get help

- **File an issue:** [github.com/camp-leatherneck/camp-leatherneck/issues](https://github.com/camp-leatherneck/camp-leatherneck/issues)
- **Ask a question:** use [Discussions](https://github.com/camp-leatherneck/camp-leatherneck/discussions)
- **Architecture docs:** [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md)
- **Personas + roles:** [`docs/PERSONAS.md`](./docs/PERSONAS.md)

## Acknowledgments

Camp Leatherneck would not exist without [Steve Yegge](https://github.com/steveyegge) and the Gas Town project. Thank you for publishing the framework under a license that makes this work possible.
