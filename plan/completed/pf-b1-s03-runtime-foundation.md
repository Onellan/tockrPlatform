# PF-B1-S03 — Runtime, persistence and presentation foundation

Status: **Implemented / terminal.**

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

1. Decide and record the owner-authorized initial one-connection SQLite policy,
### WP01 - Ordered work package
   serialized migration startup and WAL/single-instance boundary; define the
   evidence required before any later pool-width upgrade.
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

## Terminal evidence

On candidate `495d3e0278877d0f9c79fc8fb1f3e0ec65a7bf82`, the owner-authorized
initial one-connection policy resolved the inherited CTRL/IMS pool-width
conflict. The policy preserves WAL, serialized migrations and a single-instance
boundary; any later pool-width upgrade is explicitly deferred to a new
authorized, measured change.

- Independent engineering review: **PASS**; package boundaries,
  health/readiness, presentation, migration and runtime target contracts were
  reviewed read-only.
- Independent tester acceptance: **PASS** for all four acceptance criteria.
- Exact-candidate evidence: `python scripts/validate.py run full/local`,
  `python scripts/audit_codebase.py`, all active/completed plan-routing checks,
  delivery-contract tests and `git diff --check`.
- Runtime, migration, unit, integration, race and container profiles remain
  `NOT_APPLICABLE` where their prerequisites are not introduced by this
  planning Slice; no runtime implementation is claimed.
- No CTRL/IMS code, data, migration or authority was changed.

## Dependencies

PF-B1-S01 and PF-B1-S02.

## Stop/go conditions

Stop if the pool decision is guessed, if a target requires shared product
runtime dependencies, or if implementation scope expands into identity behavior.

## Rollback

Revert the architecture decision and leave the source conflict visible; do not
delete any database or runtime artifact.
