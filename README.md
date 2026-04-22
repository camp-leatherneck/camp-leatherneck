# Camp Leatherneck

**Marine-themed multi-agent workspace manager for Claude Code and friends.** A layered fork of [Gas Town](https://github.com/steveyegge/gastown) by Steve Yegge.

> 🚧 **v0.1.0 in development.** Fork established; rename and packaging underway.

## What is this?

Camp Leatherneck is a fork of Gas Town that:

- **Swaps the role vocabulary** to Marine Corps ranks (Mayor → **LT**, Deacon → **Top**, Refinery → **Gunny**, Witness → **Sarge**, Boot → **Fire Watch**)
- **Adds two first-class standing roles** not in Gas Town: **RTO** (Radio Telephone Operator — maintains a live sitrep) and a full **Fire Watch** directive (the watchdog-of-the-watchdog)
- **Ships 20 specialist directives** (Coach, QRF, Marksman, Recon, Sapper, Snoop, Doc, Brush, Scribe, Gun Bunny, Box-kicker, Ground Guide, House Mouse, Polecat, and the five standing roles)
- **Bakes in a Self-Correction Loop** — a meta-directive for how LT encodes its own mistakes into durable rules
- **Ships a Diagnostic Discipline doctrine** — four rules that counter the "anomaly → worst-case narrative → alarm" reflex with "anomaly → verify → hypothesize → test → alarm only if confirmed"
- **Distributes a Sitrep generator** — a launchd-scheduled script that synthesizes `gt`/`bd` state into a <30-second read

If you already use Gas Town, Camp Leatherneck is a drop-in replacement with richer persona semantics and stronger self-correction tooling. If you're new to the Gas Town world, both projects' docs apply.

## Relationship to Gas Town

Camp Leatherneck is a **layered fork** (Fork-C in our planning docs). We rename the user-facing surface — binary name, display strings, status-left defaults, prime banners — but leave Gas Town's internal Go identifiers untouched so upstream merges stay trivial. Steve's core engineering is unchanged.

**Credit where it's due:** Rigs, polecats, beads, convoys, the merge queue, Dolt integration, witness/deacon/refinery daemons, tmux orchestration, and the entire agent lifecycle model are Steve Yegge's inventions. Camp Leatherneck is a persona + doctrine overlay on top of that foundation, compiled into its own distribution for UX clarity.

See [NOTICE](./NOTICE) for the full attribution and [LICENSE](./LICENSE) for MIT terms.

## Install

_Coming in v0.1.0._ Planned:

```bash
brew install camp-leatherneck/tap/lt
```

The install lays down the Marine directives, the RTO launchd plist, and the Sitrep.app bundle. An `lt migrate` subcommand converts existing `~/gt/` installs in place.

## Status

| Component | Status |
|---|---|
| Fork established (`camp-leatherneck/camp-leatherneck`) | ✅ |
| NOTICE + LICENSE attribution | ✅ |
| Module path rename | 🚧 |
| Binary rename `gt` → `lt` | 🚧 |
| Embedded directives + Fire Watch / RTO roles | 🚧 |
| Homebrew tap | 🚧 |
| Documentation suite | 🚧 |
| v0.1.0 release | 🚧 |

## Upstream sync policy

We track [Gas Town upstream](https://github.com/steveyegge/gastown) and pull improvements. If a feature ships in Gas Town that we want, we merge upstream, rebuild our persona layer on top, and cut a new release. Year 1 policy: sync at minimum monthly.

## License

MIT. See [LICENSE](./LICENSE). Both Gas Town and Camp Leatherneck are MIT-licensed.

## Acknowledgments

Camp Leatherneck would not exist without [Steve Yegge](https://github.com/steveyegge) and the Gas Town project. Thank you for publishing the framework under a license that makes this work possible.
