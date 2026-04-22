---
title: Camp Leatherneck Personas
audience: Operators dispatching agents; contributors adding a new role
last-updated: 2026-04-22
author: Scribe
---

# Personas

Camp Leatherneck defines 23 personas. Each is a Marine-themed identity mapped onto a Camp Leatherneck framework role. Personas set voice, scope, and duties; the underlying role-slot (e.g., `mayor`, `deacon`, `refinery`) determines how the engine invokes the agent.

Per-role doctrine lives in [`directives/*.md`](../directives/) in this repo. This file is the roster and the tier model.

## The tiers

Personas fall into five tiers, by scope and lifetime:

1. **Commanding Officer** — human. One per deployment. The Overseer.
2. **Singular per town** — standing identity, one running instance across the whole deployment (LT, Top, Fire Watch, RTO). Running two of these against the same Camp Leatherneck install creates split state.
3. **Singular per rig** — standing identity, one per rig (Gunny, Sarge). Different rigs each have their own.
4. **Standing identity per-rig, ephemeral session** — Polecats. Their CV is durable; each session is fresh.
5. **Ephemeral specialists** — spawned on demand, stateless, safe to run many in parallel across terminals.
6. **Attached asset** — Dogs and embedded human Crew. Outside the rank chain.

The concurrency rule: one LT, one Top, one Fire Watch, one RTO per town; one Gunny and one Sarge per rig. Everyone else parallelizes freely.

## Full roster

| # | Persona | MCR Rank | Framework Role | MOS / Description | Scope | Lifetime | Spawned By |
|---|---|---|---|---|---|---|---|
| 1 | **Overseer (Joey)** | Col (O-6) / CO | N/A | Base commander | Global | Human | N/A |
| 2 | **LT** | 2ndLt (O-1) | `mayor` | Junior officer / Chief of Staff | Town | Standing (1/town) | Overseer |
| 3 | **Top** | 1stSgt (E-8) | `deacon` | Senior enlisted / town watchdog | Town | Standing (1/town) | Fire Watch / Overseer |
| 4 | **Gunny** | GySgt (E-7) | `refinery` | Senior NCO / merge gate | Rig | Standing (1/rig) | LT / Top |
| 5 | **Marksman** | SSgt (E-6) | specialist | Scout Sniper (0317) — security review | On-demand | Ephemeral | LT / Top |
| 6 | **Coach** | SSgt (E-6) | specialist | Primary Marksmanship Instructor — QA & tests | On-demand | Ephemeral | LT / Top |
| 7 | **Recon** | Sgt (E-5) | specialist | Marine Recon (0321) — competitive intel | On-demand | Ephemeral | LT / Top |
| 8 | **Sapper** | Sgt (E-5) | specialist | Combat Engineer (1371) — CI/CD & IaC | On-demand | Ephemeral | LT / Top |
| 9 | **Snoop** | Sgt (E-5) | specialist | Intel Analyst (0231) — non-adversarial research | On-demand | Ephemeral | LT / Top |
| 10 | **Sarge** | Sgt (E-5) | `witness` | Squad leader — polecat lifecycle on rig | Rig | Standing (1/rig) | LT / Top |
| 11 | **QRF** | Sgt (E-5) | specialist | Quick Reaction Force — prod incidents | On-demand | Ephemeral | Top / LT |
| 12 | **RTO** | Sgt (E-5) | specialist (standing) | Radio Telephone Operator (0621) — sitrep | Town | Standing (cron+event) | Overseer / LT |
| 13 | **Doc (Devil Doc)** | HM2 (E-5, Navy) | specialist | Hospital Corpsman — broken-code triage | On-demand | Ephemeral | LT / Top |
| 14 | **Brush** | Cpl (E-4) | specialist | Combat Artist (4671) — visual design | On-demand | Ephemeral | LT |
| 15 | **Scribe** | Cpl (E-4) | specialist | Combat Correspondent (4341) — docs | On-demand | Ephemeral | LT / Top |
| 16 | **Gun Bunny** | Cpl (E-4) | specialist | Field Artillery Cannoneer (0811) — performance | On-demand | Ephemeral | LT / Top |
| 17 | **Box-kicker** | Cpl (E-4) | specialist | Warehouse/Supply (3043) — deps | On-demand | Ephemeral | LT / Top |
| 18 | **Ground Guide** | LCpl (E-3) | specialist | Pre-flight / deploy walk-through | On-demand | Ephemeral | LT |
| 19 | **Polecat** | LCpl (E-3) | `polecat` | Rifleman (0311) — execute slung beads | Rig | Standing CV / ephemeral session | LT |
| 20 | **Crew** | Civilian | `crew` | Human crew embedded on a rig | Rig | Standing | LT |
| 21 | **House Mouse** | PFC (E-2) | specialist | Junior Marine — barracks detail / cleanup | On-demand | Ephemeral | Top / LT |
| 22 | **Fire Watch** | Pvt (E-1) | `boot` | Boot recruit — keeps Top alive | Town | Standing (1/town) | System |
| 23 | **Dog** | MWD (attached) | `dog` | Military Working Dog — infra tasks | Cross-rig | Ephemeral | Top |

## Role-slot ↔ persona mapping

Camp Leatherneck's engine invokes roles by their framework role name. The voice and doctrine the user sees is the Camp Leatherneck persona bound to that slot:

| Camp Leatherneck role-slot | Camp Leatherneck persona |
|---|---|
| `mayor` | LT |
| `deacon` | Top |
| `refinery` | Gunny |
| `witness` | Sarge |
| `boot` | Fire Watch |
| `polecat` | Polecat |
| `crew` | Crew |
| `dog` | Dog |

`gt mayor attach` still works — it attaches you to the LT session. Internal Go identifiers and CLI subcommand names are upstream's and unchanged; only the display strings and directive voice come from the overlay.

## Specialists are safe to parallelize

Marksman, Coach, Recon, Sapper, Snoop, QRF, Doc, Brush, Scribe, Gun Bunny, Box-kicker, Ground Guide, House Mouse — all ephemeral, all safe to run multiple concurrent instances across terminals. Each spawns with a fresh directive prime; no persistent identity.

RTO is the exception among Sgt-tier specialists: one per town, because its duty is to maintain the single-file sitrep.

Polecats are a hybrid: standing CV, ephemeral session. You can have many polecats on a rig (one per worktree), but each polecat identity is singular. Polecats are **blocked** from personal-domain tools (Gmail, Messages, Calendar, Notes, Contacts, Reminders, Drive, Docs) — misrouted beads bounce back to LT.

## Where the per-role doctrine lives

Every persona with a directive file has it at `directives/<name>.md` in this repo. Read the directive at mission time before spawning, not from memory. The directive carries voice, output format, effort caps, and anti-patterns specific to that role.

Fire Watch and Dog have no directive file — they are narrow binary-level roles whose behavior is fully defined by the Camp Leatherneck engine. LT and Top dispatch everything else.

## See also

- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — where personas sit in the fork
- [`SELF-CORRECTION.md`](./SELF-CORRECTION.md) — LT's meta-loop, applies to every standing role
- [`DIAGNOSTIC-DISCIPLINE.md`](./DIAGNOSTIC-DISCIPLINE.md) — rules every watchdog-adjacent persona follows
- [`directives/`](../directives/) — per-role doctrine
- Upstream [`README.md`](../README.md) — engine-side role-slot reference
