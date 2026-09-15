# PF-B1-S03 — Runtime, persistence and presentation foundation

Status: **Blocked / NOT RUN — Platform measurement prerequisite unavailable.**

## Objective

Turn the approved contracts into an implementation-ready runtime shape while
resolving the inherited SQLite pool conflict and defining health, container and
presentation seams without implementing Platform behavior prematurely.

## Authority and current evidence

Authority is `architecture.md`, the presentation/security/migration contracts
and the CTRL/IMS source-alignment conflict record. CTRL describes one SQLite
connection; IMS records measured file-backed `4/2` pooling after migrations.

## Affected files/packages

Future `go.mod`, `cmd/platform`, `internal/*`, `web/*`, Docker/runtime files and
health/readiness contracts; current Slice updates architecture and plan indexes
only until implementation authority is separately invoked.

## Ordered work

1. Decide and record the measured initial SQLite connection policy, serialized
### WP01 - Ordered work package
   migration startup and WAL/single-instance boundary.
Route: kind=other; risk=H[DATA,CONC,OPS]
2. Freeze package dependency rules, `/healthz`/`/readyz` semantics, graceful
### WP02 - Ordered work package
   shutdown and bounded HTTP resources.
Route: kind=other; risk=H[OPS,DEPLOY,API]
3. Freeze templ/Tailwind/source-owned primitive and hardened container targets;
### WP03 - Ordered work package
   identify prerequisites without adding runtime code in planning.
Route: kind=other; risk=H[UI,DEPLOY,DEP]

## Migration impact

None in planning. The future first migration is `platform.db` ledger bootstrap;
it must not open or rewrite CTRL/IMS databases.

## Security impact

Health endpoints must not disclose secrets or tenant data. Container target is
non-root, capability-dropped, read-only root with bounded `/tmp` and persistent
data.

## Acceptance criteria

- the SQLite conflict has an evidence-based owner decision or explicit blocked
  stop/go record;
- package boundaries and dependency direction are testable;
- health/readiness, runtime targets and presentation rules are explicit;
- no ORM, service mesh, broker, shared database or runtime Node dependency is
  introduced.

## Tests and evidence

Architecture, frontend and quality profiles; independent architecture review of
the package graph and runtime contract; no build claim until a Dockerfile and Go
module exist in a later implementation candidate.

## Current execution outcome

On candidate `2f27ca5eed2cc0af12111c9ea1623af23a5df3b4`, independent review and
tester acceptance classified the Slice as **BLOCKED / NOT RUN**, not terminal:

- AC02 package boundaries and dependency direction: **PASS** from the target
  architecture package map.
- AC03 health/readiness, runtime targets and presentation rules: **PASS** from
  the security-runtime, presentation and source-alignment contracts.
- AC04 prohibited ORM/service-mesh/broker/shared-database/runtime-Node targets:
  **PASS** from the architecture and presentation contracts.
- AC01 SQLite pool-width owner decision: **BLOCKED / NOT RUN**. CTRL records a
  single connection and IMS records a measured file-backed `4/2` pool, while
  Platform has no `go.mod`, runtime package or measurement harness. Choosing a
  width now would be an unsupported guess.

Resume only when an authorised Platform measurement prerequisite exists and
the pool-width decision can be bound to fresh evidence. Do not add runtime code,
choose a pool width, or certify PF-B1 while this blocker remains.

## Dependencies

PF-B1-S01 and PF-B1-S02.

## Stop/go conditions

Stop if the pool decision is guessed, if a target requires shared product
runtime dependencies, or if implementation scope expands into identity behavior.

## Rollback

Revert the architecture decision and leave the source conflict visible; do not
delete any database or runtime artifact.
