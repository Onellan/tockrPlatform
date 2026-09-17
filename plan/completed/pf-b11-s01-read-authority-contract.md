# PF-B11-S01 — Read-authority contract and consumer compatibility

**Status:** Implemented / terminal
**Priority:** PF — Platform consumer read-authority extension
**Batch:** PF-B11
**Objective:** Freeze the versioned contract that makes Platform-backed local
consumer projections authoritative without introducing synchronous
per-request Platform reads.

## Authority and scope

This Slice was authorised by the PF-B11 forward extension and the Platform
ownership, v1 contract, events/projection, coding standards and validation
authorities listed in the original active plan. It adds the separate
`platform.read-authority.v1` contract and compatibility matrix only. It does
not implement durable snapshots, HTTP feed routes, consumer code, product
roles, billing, shared storage or authority cutover.

## Delivered scope

- [`platform-read-authority-v1.md`](../../docs/contracts/platform-read-authority-v1.md)
  freezes the entity/relationship allow-list, exact snapshot/feed/status
  envelopes, provenance, opaque cursor/checksum rules, fail-closed states,
  limits, error taxonomy, Ed25519 request signing, replay/rotation behavior,
  compatibility and rollback.
- [`platform-read-authority-compatibility-v1.md`](../../docs/contracts/platform-read-authority-compatibility-v1.md)
  freezes the identical CTRL/IMS consumer protocol, product pairing and
  bootstrap/change negative behavior without transferring product authority.
- `internal/platform/readauthority` provides the reusable version, route,
  header, consumer/product, state, bounded-request, canonical-signature and
  per-entity record validation seam. It has no persistence or HTTP runtime.
- The canonical unit validation route now executes the focused contract
  package.

## Ordered work packages

### WP01 - Contract delta

Route: kind=other; risk=H[API,AUTH,DATA,DEP,DOC]

### WP02 - Wire and compatibility contract

Route: kind=authorization; risk=H[AUTH,API,DEP,GOV,DOC]

### WP03 - Staff Engineer plan/design review

Route: kind=other; risk=H[AUTH,CONC,API,DEP,GOV,DOC]

## Candidate-bound evidence

Accepted implementation candidate: `b79b9321a06dd1c0e25381127dc61e861bae520d`.

- WP01 contract delta: **PASS**.
- WP02 wire/compatibility contract: **PASS**.
- WP03 Staff Engineer plan/design review: **PASS TO IMPLEMENTATION** before
  implementation; independent engineering review of the implementation:
  **PASS** after the repair of candidate `33fa0c5`.
- Independent tester acceptance: **PASS**, with evidence IDs S01-TEST-E01
  through S01-TEST-E03, bound to the accepted candidate.
- Exact-candidate local validation on the accepted implementation candidate:
  format, architecture, security, unit and plan-routing checks **PASS**.
- The initial unsupported `validate.py go-test` attempt was classified
  `INVOCATION_FAIL`; the repository-resolved unit profile then executed the
  focused package successfully. No application repair was based on that
  context failure.

## Boundary and sequencing

No CTRL/IMS production code, data, migration, product role, billing fact,
shared database or authority cutover was changed. PF-B11-S02 was not started
until this Slice was reviewed, accepted, locally validated and terminally
closed.

PF-B11-S01 is **PASS / terminal**. PF-B11-S02 is promoted to Ready by the
PF-B11 queue; the PF-B11 Batch remains non-terminal until S02, S03 and S04 are
completed and certified.
