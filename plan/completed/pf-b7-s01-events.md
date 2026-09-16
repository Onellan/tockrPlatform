# PF-B7-S01 — Platform events and transactional outbox

Status: **Implemented / terminal**

## Objective

Implement a versioned, durable Platform event envelope and outbox for shared
identity/access changes without creating product-domain events or a broker
dependency.

## Authority and current evidence

Authority is `docs/architecture/events-and-projection-contract.md` and
contract v1. CTRL/IMS PD plans call for versioned API/events and bounded local
projections, not a shared database.

## Affected files/packages

`internal/events`, outbox domain/store/SQLite files, transaction composition,
event serialization, audit and tests.

## Ordered work

1. Define event types, aggregate sequences, schema versions and redaction rules.
### WP01 - Ordered work package
Route: kind=other; risk=H[API,HIST,DOC]
2. Write outbox records atomically with shared-authority mutations and publish
### WP02 - Ordered work package
   only committed facts.
Route: kind=migration; risk=H[DATA,CONC,HIST]
3. Prove duplicate-safe event IDs, bounded payloads and no secret/product-role
### WP03 - Ordered work package
   leakage.
Route: kind=other; risk=H[AUTH,API,OPS]

## Migration impact

New outbox/event tables use ordered migrations with fresh, upgrade, reopen and
divergence tests. Existing product event history is not imported.

## Security impact

Events contain no credentials, raw session tokens, product roles or billing.
Access to event reads is server-side and payloads are least-knowledge.

## Acceptance criteria

Committed Platform authority changes yield one versioned outbox fact; failed
transactions yield none; consumers can identify duplicates and sequences.

## Tests and evidence

Transactional failure/commit tests, serialization allow-list, migration suite,
concurrency/race evidence and independent review.

## Dependencies

PF-B5-S02 and PF-B1-S03.

## Stop/go conditions

Stop if event delivery requires Kafka/Redis, if outbox facts can precede commit,
or if the payload becomes product-domain authority.

## Rollback

Stop publication and replay from durable outbox after repair; retain committed
facts and sequence history.

## Terminal evidence

Accepted implementation candidate: `bde43056446327103f8e1241ce460aebe561bdcd`.

Terminal closeout candidate: `7ad1b8988fe81a0767b543fd09927a2d277a5e02`.

- Independent engineering review: **PASS**; no R1 finding remains.
- Independent tester acceptance: **PASS** for versioned envelope allow-list,
  transaction atomicity, migration safety, bounded delivery and redaction.
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
[`PF-B7-S01 reconciliation`](../../docs/implementation/audits/pf-b7-s01-events.md),
[`independent review`](../../docs/implementation/audits/pf-b7-s01-independent-review.md),
[`independent acceptance`](../../docs/implementation/audits/pf-b7-s01-independent-acceptance.md).
