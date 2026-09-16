# PF-B10-S02 — Final Platform foundation certification

Status: Ready

## Objective

Certify the exact Platform candidate and all PF acceptance boundaries, then
record the foundation programme as ready for subsequent authorised runtime
delivery without declaring CTRL/IMS migration complete.

## Authority and current evidence

Authority is PF, all terminal Batch plans, local validation contract and
`docs/implementation/IMPLEMENTED.md`. The exact candidate SHA must be bound to
all evidence; remote CI is not a substitute for local proof.

## Affected files/packages

Certification ledgers, final architecture/security review, acceptance evidence,
implementation index and terminal delivery record. No product runtime scope is
expanded.

## Ordered work

1. Reconcile all Batch/Slice acceptance rows, dependencies, review findings and
### WP01 - Ordered work package
   stop/go decisions against one candidate.
Route: kind=other; risk=H[DOC,GOV]
2. Run complete `full/local` plus required browser/container/architecture and
### WP02 - Ordered work package
   cross-contract evidence; classify every result.
Route: kind=other; risk=H[AUTH,DATA,UI,OPS,DEPLOY]
3. Record terminal PF evidence and explicit next Ready Slice only if every gate
### WP03 - Ordered work package
   passes.
Route: kind=other; risk=H[GOV,HIST,DOC]

## Migration impact

Certification must prove no CTRL/IMS production data or authentication was
migrated or changed. Any migration tooling remains fixture-only unless separately
authorized.

## Security impact

No blocked security, authorization, history, runtime or contract evidence may be
converted to PASS. Secret scans and exact candidate binding are mandatory.

## Acceptance criteria

Every PF Slice has independent review/acceptance evidence, every Batch has full
certification evidence, local profiles pass or are explicitly not applicable
with authority, and the repository is ready for PF-B1 runtime implementation.

## Tests and evidence

`python scripts/validate.py run full/local`, exact-candidate fingerprint,
independent final architecture review, tester ledger, browser/runtime/container
evidence and publication record.

## Dependencies

PF-B10-S01 and all prior Batch terminal gates.

## Stop/go conditions

Stop on any missing acceptance row, changed candidate, blocked required profile,
unresolved source conflict or unauthorized external migration.

## Rollback

Do not mark terminal completion. Preserve the candidate and evidence, return to
the owning Slice repair loop and rerun affected independent gates.
