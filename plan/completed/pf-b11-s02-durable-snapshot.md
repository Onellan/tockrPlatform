# PF-B11-S02 — Durable snapshot and source cursor

**Status:** Implemented / terminal
**Priority:** PF — Platform consumer read-authority extension
**Batch:** PF-B11
**Depends on:** PF-B11-S01-R1 terminal

## Objective

Implement the Platform-side immutable bootstrap snapshot and source-cursor
seam defined by `platform.read-authority.v2`, using capability-local domain,
store and SQLite modules without exposing SQLite representation to callers.

## Authority and delivered scope

PF-B11-S01-R1 terminally publishes `platform.read-authority.v2`, which keeps
v1 event provenance unchanged and adds explicit `migration_seed` provenance.
This Slice implemented only the Platform snapshot/source-cursor capability:

- persistence-neutral `internal/domain` and narrow `internal/store` snapshot
  contracts;
- ordered migration 10 for consumer-bound immutable snapshot metadata and
  allow-listed records, including ordinal, duplicate, hash and completion
  invariants;
- one-transaction materialization from the canonical tables and outbox
  boundary, with deterministic canonical ordering and SHA-256 checksum;
- v2 event provenance for committed records and migration 6
  version/name/checksum provenance for the two pre-outbox Product seeds;
- bounded paging with snapshot/consumer/entity binding, expiry and bounded
  retention cleanup; and
- corruption, omission, duplicate, missing-provenance, expiry, source
  mutation, migration and reopen evidence.

No terminal v1 event payload, source authority row, product role, billing fact,
CTRL/IMS runtime, feed route, shared database or authority cutover changed.

## Ordered work and acceptance

### WP01 — Deep read-authority seam

Route: kind=other; risk=H[DATA,API,DEP,DOC]

The domain/store seam exposes validated snapshot metadata and records without
SQLite table knowledge, product-role knowledge or billing knowledge.

### WP02 — Snapshot migration

Route: kind=migration; risk=H[DATA,HIST,CONC,OPS]

Migration 10 creates immutable metadata and record tables under the existing
single-connection SQLite policy. Completion is promoted only after all rows
and the checksum are committed in one transaction.

### WP03 — Consistent materialization

Route: kind=migration; risk=H[DATA,CONC,API,PERF]

The source cursor and all selected canonical records are read from one
transaction. Records are bounded, ordered by the v2 canonical kind/identity
order, validated at the contract seam and checksum-bound before finalization.

### WP04 — Retention and integrity evidence

Route: kind=other; risk=H[DATA,CONC,OPS,PERF]

Paging, expiry, cleanup, source mutation immutability, duplicate prevention,
checksum/hash corruption, missing provenance, fresh/upgrade/reopen migration,
event/seed provenance and repository race behavior are covered.

## Candidate-bound evidence

Accepted implementation candidate:
`065564e9db4be88dc556bb4b0fd0a88050c9487a`.

- S02-AC01 through S02-AC05: **PASS**.
- Independent engineering review: **PASS**, recorded in
  [`docs/implementation/audits/pf-b11-s02-engineering-review.md`](../../docs/implementation/audits/pf-b11-s02-engineering-review.md).
- Independent tester acceptance: **PASS**, recorded in
  [`docs/implementation/audits/pf-b11-s02-tester-acceptance.md`](../../docs/implementation/audits/pf-b11-s02-tester-acceptance.md).
- The initial review R1 findings and repair are retained in
  [`docs/implementation/audits/pf-b11-s02-engineering-review-initial.md`](../../docs/implementation/audits/pf-b11-s02-engineering-review-initial.md).
- Exact-candidate local validation on the implementation candidate:
  `format`, `architecture`, `security`, `quality`, `migration`, `unit`,
  `integration` and `race` **PASS**.

No required S02 evidence is blocked or not run. Container profiles are not
required because this Slice does not change an authorised container/runtime
surface.

## Completion and sequencing

PF-B11-S02 is **PASS / terminal**. PF-B11-S03 is promoted to **Ready**.
PF-B11-S04 remains planned. The PF-B11 Batch remains non-terminal until S03,
S04 and Batch certification pass.
