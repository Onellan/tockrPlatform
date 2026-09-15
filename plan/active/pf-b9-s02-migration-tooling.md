# PF-B9-S02 — Dry-run/import and rollback tooling

Status: Planned

## Objective

Prepare an explicit, staged, recoverable import workflow from CTRL/IMS into
Platform without running it against production in PF.

## Authority and current evidence

Authority is the PF stop boundary, migration contract and reconciliation plan.
Source product plans require import → shadow → auth/read/write cutover only with
independent evidence and rollback.

## Affected files/packages

Reconciliation CLI/commands, import manifests, checkpoints, rollback/runbook
docs, fixtures and migration tests. Production data is out of scope.

## Ordered work

1. Define signed dry-run manifest, deterministic order, checkpoint and explicit
### WP01 - Ordered work package
   approval boundary.
Route: kind=migration; risk=H[DATA,GOV,HIST,OPS]
2. Implement fixture-only import and compensating rollback rehearsal; preserve
### WP02 - Ordered work package
   unknown/ambiguous records as blocked.
Route: kind=migration; risk=H[DATA,AUTH,CONC]
### WP03 - Ordered work package
3. Prove no-loss, idempotency, resume, rollback and audit evidence.
Route: kind=migration; risk=H[DATA,HIST,OPS,DEPLOY]

## Migration impact

This is the first Slice with import-shaped code, but only disposable fixtures
are allowed. No CTRL/IMS user or production record is migrated.

## Security impact

Require explicit operator authorization, least-privilege source access, secret
redaction and fail-closed checkpoint integrity.

## Acceptance criteria

Fixture import is deterministic/idempotent, ambiguous records block, rollback is
recoverable and the runbook makes production execution a separately authorized
operation.

## Tests and evidence

Fixture migration suite, fresh/upgrade/reopen Platform DB tests, crash/resume,
rollback/no-loss checks, audit and independent acceptance.

## Dependencies

PF-B9-S01, PF-B7-S02 and PF-B1-S03.

## Stop/go conditions

Stop if production data is selected, if rollback is destructive/unverified, or
if a record requires guessed identity/history.

## Rollback

Use fixture database disposal only; preserve manifests/checkpoints and do not
touch CTRL/IMS production data.
