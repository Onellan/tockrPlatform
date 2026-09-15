# PF-B1-S03 — Runtime, persistence and presentation foundation

Status: Planned

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

## Dependencies

PF-B1-S01 and PF-B1-S02.

## Stop/go conditions

Stop if the pool decision is guessed, if a target requires shared product
runtime dependencies, or if implementation scope expands into identity behavior.

## Rollback

Revert the architecture decision and leave the source conflict visible; do not
delete any database or runtime artifact.
