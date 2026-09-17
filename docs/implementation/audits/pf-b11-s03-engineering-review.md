# PF-B11-S03 independent engineering review

**Candidate:** `bfc111ad1e7f8add6967e2dbb8b41f8ac2d192ea`
**Decision:** **PASS**
**Authority:** PF-B11-S03 WP01–WP04 and S03-AC01–S03-AC06

## Review method

The repaired candidate was reviewed read-only against the S03 plan, the
published `platform.read-authority.v2` contract, Platform ownership and
boundary authority, coding standards, and the codebase-design depth and
deletion test. The review covered signed machine authentication, key overlap
and retirement, bounded timestamp and nonce replay protection, consumer
binding, body/page/response/rate limits, snapshot and feed route seams,
cursor-gap detection, source-event validation, safe errors, and redaction
boundaries.

The initial review finding is recorded in
[`pf-b11-s03-engineering-review-initial.md`](pf-b11-s03-engineering-review-initial.md).
The repaired verifier compares parsed timestamps against a bounded current
time interval, avoiding signed arithmetic overflow. The regression test
proves that an extreme timestamp is rejected.

## Findings

- Ed25519 verification is performed before the route handlers and is bound to
  the consumer, key ID, canonical method/path/query/body digest, timestamp and
  nonce.
- Active and overlap keys are deployment-managed; unknown or retired keys,
  malformed signatures, replayed nonces and cross-consumer key substitution
  fail closed. Nonces are durably reserved in SQLite with expiry cleanup.
- Body, page, response and per-consumer rate limits are explicit. Snapshot
  records are served only after S02 integrity verification; committed outbox
  events are validated and cursor gaps return resynchronisation rather than an
  inferred continuation.
- Routes are outside the browser-session group and do not use session cookies,
  product roles, a shared database, credentials, document bytes or secret
  diagnostics. The feed preserves the terminal `platform-events-v1` payload
  boundary.

No blocking review finding remains.

## Gate evidence

`format`, `architecture`, `security`, `quality` and the focused HTTP contract
tests passed on the repaired candidate. The repository-native runtime profiles
are required for tester acceptance and exact-candidate local validation.

