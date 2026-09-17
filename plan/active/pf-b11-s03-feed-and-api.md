# PF-B11-S03 — Authenticated feed and resynchronisation API

**Status:** Planned
**Priority:** PF — Platform consumer read-authority extension
**Batch:** PF-B11
**Depends on:** PF-B11-S02 terminal

## Objective

Expose the bounded machine-to-machine read-authority transport defined by
S01/S01-R1 v2 and backed by S02. The API supplies a verified bootstrap snapshot, ordered
committed changes and explicit resynchronisation/status outcomes; it never
becomes a synchronous product request dependency.

## Required transport surface

The implementation must provide the S01 wire contract for:

- `POST /api/v1/read-authority/snapshots` — create a bounded consumer-bound
  snapshot and return its immutable metadata;
- `GET /api/v1/read-authority/snapshots/{snapshotID}/records` — page finalized
  allow-listed records by entity kind and opaque page cursor;
- `GET /api/v1/read-authority/changes` — page committed changes after an
  opaque global cursor, with `next_cursor`, bounded limits and explicit
  `resync_required`; and
- `GET /api/v1/read-authority/status` — return source/contract readiness and
  freshness state without returning sensitive records.

The exact JSON schemas, headers, limits and error classes are authoritative in
the v2 contract. Snapshot records must preserve event or migration-seed
provenance; the incremental feed remains `platform-events-v1`. These endpoints
are not browser-session routes and do not
grant product access.

## Ordered work

### WP01 - Machine authentication

Add per-consumer deployment-managed Ed25519 public-key configuration, key-ID
overlap/rotation and a capability-local signed-request verifier with bounded
timestamp and durable nonce replay protection.

Route: kind=authorization; risk=H[AUTH,CONC,OPS,API,DEP]

### WP02 - Read-authority routes

Add the integration routes and response headers, enforce consumer/product
binding, body/response/page/rate limits, safe error classification and
secret-free diagnostics; keep browser cookies and user assertions out of the
machine boundary.

Route: kind=authorization; risk=H[AUTH,API,OPS,DEP]

### WP03 - Snapshot and change feed

Expose finalized snapshot pages and committed outbox changes through the
opaque cursor contract, detect cursor expiry/gaps and return explicit
resynchronisation/status outcomes without silently skipping events.

Route: kind=other; risk=H[DATA,CONC,API,PERF,OPS]

### WP04 - Adversarial transport evidence

Add HTTP contract, adversarial authorization, replay, rotation, resource
limit, cursor-gap, stale/unavailable and response-redaction tests for both
`tockrctrl` and `tockrims` compatibility fixtures.

Route: kind=authorization; risk=H[AUTH,DATA,API,OPS,DEP]

## Risk profile

`AUTH DATA CONC OPS API PERF DEP` are high. `HIST` is high for cursor and
provenance continuity. There is no UI scope.

## Acceptance criteria

- **S03-AC01:** only a valid consumer signature with an active configured key,
  acceptable timestamp and unused nonce reaches the read-authority handlers.
- **S03-AC02:** key rotation accepts the documented overlap and rejects
  retired/unknown keys, malformed signatures, replays and cross-consumer
  product keys.
- **S03-AC03:** v2 snapshots and `platform-events-v1` changes are bounded,
  deterministic, committed, redacted and version-checked; event or
  migration-seed provenance is preserved and partial, stale, blocked,
  unavailable or gapped state cannot be returned as authoritative.
- **S03-AC04:** cursor expiry, source retention loss and ordering gaps return
  `resync_required` rather than an inferred continuation.
- **S03-AC05:** the endpoints have no browser-session fallback, no per-request
  product authorization role evaluation and no shared-database path.
- **S03-AC06:** security, HTTP/resource and contract tests pass for both
  consumer compatibility profiles, including negative and no-secret-log
  evidence.

## Evidence and validation intent

Use `unit`, `integration`, `security`, `architecture` and `race` profiles,
plus focused HTTP contract tests. Independent tester acceptance must exercise
both consumer identities and prove that an authenticated Platform user cannot
substitute for a machine consumer signature.

### TestContext and preflight

```text
candidate=<exact S03 implementation candidate under review>
authority=PF-B11-S03-AC01..AC06
surface=AUTH|API|DATA|CONC|OPS|DEP
profile=unit,integration,security,architecture,race plus focused HTTP tests
command_source=validation_registry.py and validate.py
preflight=PASS required before interpreting any behavioral result
```

Preflight must prove that both consumer compatibility selectors resolve to
real tests. Zero matches is `INVOCATION_FAIL`; unavailable tools, fixtures
or environment are `BLOCKED / NOT RUN`.

## Stop/go and rollback

Stop if request signing cannot be made replay-resistant, if the endpoint can
return a partial snapshot, or if feed recovery can skip a cursor. Rollback is
an explicit disablement of the new integration routes/configuration while
preserving canonical Platform mutations, outbox rows and existing assertion /
projection behavior.

## Completion

Move to `plan/completed/` only after exact-candidate implementation,
independent review, independent acceptance and local validation pass. S04 must
then certify the complete Batch before the consumer gates are eligible for
promotion.
