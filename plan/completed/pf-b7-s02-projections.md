# PF-B7-S02 — Projection inbox and reconciliation support

Status: **Implemented / terminal**

## Objective

Provide idempotent local projection/inbox support for Platform events and
explicit stale/gap/reconciliation states.

## Authority and current evidence

Authority is the events/projection contract and CTRL/IMS PD resilience plans.
Products remain owners of their local projections and product authorization.

## Ordered work

1. Define inbox identity, per-aggregate sequence, gap and stale-state model.
### WP01 - Ordered work package
Route: kind=migration; risk=H[DATA,CONC,API]
2. Implement idempotent apply/replay boundaries with bounded batch work.
### WP02 - Ordered work package
Route: kind=other; risk=H[DATA,OPS,PERF]
3. Prove duplicate, out-of-order, missing and unavailable cases remain explicit
### WP03 - Ordered work package
   and fail closed for security-sensitive reads.
Route: kind=authorization; risk=H[AUTH,CONC,OPS]

## Migration impact

Inbox/projection migrations require fresh/upgrade/reopen and replay fixtures;
unknown source versions are retained as blocked, not discarded.

## Security impact

Stale projections cannot silently grant access. Reconciliation data is scoped,
redacted and audited.

## Acceptance criteria

Duplicate events are harmless, gaps are detected, replay is bounded and
security-sensitive consumers distinguish current/stale/unavailable state.

## Tests and evidence

Idempotency/order/replay tests, migration tests, failure/recovery evidence,
performance budget and independent tester acceptance.

## Dependencies

PF-B7-S01 and PF-B6-S02.

## Stop/go conditions

Stop if eventual state is used as unconditional authorization or if events are
silently dropped to make a projection appear current.

## Rollback

Pause projection consumption, preserve inbox/outbox state and replay after a
corrective version; no destructive truncation.

## Terminal evidence

Accepted implementation candidate: `07c2b23ac35645b809fea3b1fc87042932f111b9`.

Terminal closeout candidate: `5d884cc3395e7ee6b2b7a11010f7bac3f435b8ba`.

- Independent engineering review: **PASS**; no R1 finding remains.
- Independent tester acceptance: **PASS** for duplicate identity, ordered
  apply, gap detection, bounded reconciliation, blocked unknown versions,
  unavailable recovery and migration safety.
- Exact-candidate local validation: format, architecture, security, migration,
  frontend, quality, unit, integration and extended repository-wide race
  evidence **PASS**. The repository composite `full/local` race child exceeded
  its fixed 300-second timeout; that diagnostic is retained as `TIMEOUT`, not
  reported as a pass.
- Container build profiles: **NOT_APPLICABLE** because no authorised
  Dockerfile exists.
- The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  production code, data, migration, product role or authority cutover was
  changed.

Detailed evidence:
[`PF-B7-S02 reconciliation`](../../docs/implementation/audits/pf-b7-s02-projections.md),
[`independent review`](../../docs/implementation/audits/pf-b7-s02-independent-review.md),
[`independent acceptance`](../../docs/implementation/audits/pf-b7-s02-independent-acceptance.md).
