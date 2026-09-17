# PF-B11-S02 — Durable snapshot and source cursor

**Status:** Ready
**Priority:** PF — Platform consumer read-authority extension
**Batch:** PF-B11
**Depends on:** PF-B11-S01-R1 terminal

## Objective

Implement the Platform-side immutable bootstrap snapshot and source-cursor
seam defined by `platform.read-authority.v2`, using capability-local domain,
store and SQLite modules without exposing SQLite representation to callers.

## Current-state evidence

Platform currently owns canonical current tables and an outbox, but its
projection support stores inbox/checkpoint state only. There is no durable,
consistent snapshot that a consumer can verify and page through while source
mutations continue.

## Resolved authority evidence

PF-B11-S01-R1 terminally publishes `platform.read-authority.v2`, which keeps
v1 event provenance unchanged and adds an explicit `migration_seed` form. The
two active Product rows seeded by migration 6 must carry the recorded
migration version, name and checksum; they must not receive synthetic `evt_*`
identities, omitted records or fabricated Product-created events. Later Product
mutations remain committed `platform-events-v1` changes. S02 may now proceed
against the v2 contract without changing the terminal event allow-list.


## Affected surfaces

- `internal/domain/platform_read_authority.go` for validated snapshot,
  record, cursor and state types;
- `internal/store/platform_read_authority.go` for narrow snapshot/source
  contracts;
- capability-local SQLite schema/migration and implementation files for
  immutable snapshot metadata/rows, source cursor and retention;
- existing outbox read seam, without changing the v1 event payload allow-list;
- migration tests and deterministic fixture/contract tests.

## Ordered work

### WP01 - Deep read-authority seam

Define the deep read-authority domain/store seam and prove that callers need no
SQLite table knowledge, product role knowledge or billing knowledge.

Route: kind=other; risk=H[DATA,API,DEP,DOC]

### WP02 - Snapshot migration

Add an ordered migration for immutable snapshot metadata and allow-listed rows,
including consumer binding, contract version, source cursor, checksum,
creation/expiry, deterministic ordinal and terminal status.

Route: kind=migration; risk=H[DATA,HIST,CONC,OPS]

### WP03 - Consistent materialization

Materialize a bounded snapshot from one consistent SQLite source view,
calculate a canonical checksum and expose only complete/finalized snapshots;
never publish partial rows as current authority.

Route: kind=migration; risk=H[DATA,CONC,API,PERF]

### WP04 - Retention and integrity evidence

Add bounded retention/cleanup and source-cursor tests, including duplicate
record prevention, expiry, checksum mismatch, concurrent source mutation,
fresh/upgrade/reopen migration behavior and deterministic ordering.

Route: kind=other; risk=H[DATA,CONC,OPS,PERF]

## Risk profile

`DATA HIST CONC OPS API PERF DEP` are high. `AUTH` remains high at the
consumer-binding and allow-list boundary. No consumer data or product-domain
tables may be copied into Platform.

## Acceptance criteria

- **S02-AC01:** snapshot rows contain only the S01 allow-list and preserve
  canonical IDs, relationship integrity and v2 event or migration-seed
  provenance.
- **S02-AC02:** a snapshot is atomically complete, checksum-bound,
  deterministic and bound to a source cursor; partial/expired/corrupt data is
  never served as current.
- **S02-AC03:** snapshot creation and paging are bounded and safe under source
  mutation; limits, retention and cleanup cannot cause silent record loss.
- **S02-AC04:** the migration passes fresh, upgrade and reopen evidence and
  does not alter terminal PF schema meaning or one-connection policy.
- **S02-AC05:** unit, SQLite integration, race/concurrency and adversarial
  integrity tests prove duplicate, omission, checksum, expiry and scope
  failures are explicit and fail closed.

## Evidence and validation intent

Use the registry `migration`, `unit`, `integration`, `race`, `architecture`
and `security` profiles as applicable. Independent review must apply the
codebase-design depth/deletion test to the new seam and verify that no generic
repository/service layer was introduced.

### TestContext and preflight

```text
candidate=<exact S02 implementation candidate under review>
authority=PF-B11-S02-AC01..AC05
surface=DATA|HIST|CONC|OPS|API|DEP
profile=migration,unit,integration,race,architecture,security
command_source=validation_registry.py and validate.py
preflight=PASS required before interpreting any behavioral result
```

Fresh/upgrade/reopen fixtures and focused selectors must be preflighted. A
fixture, environment, tool or prerequisite failure is context evidence, not a
product failure.

## Stop/go and rollback

**Ready:** the provenance authority conflict is resolved by the terminal R1 v2
contract. Implement only the S02 snapshot/source-cursor scope; do not change
the terminal v1 event payload contract or delete source authority rows.

## Completion

Move to `plan/completed/` only after exact-candidate implementation,
independent review, independent acceptance and local validation pass. S03 may
not begin while snapshot integrity or migration evidence is incomplete.
