# PF-B11-S01 — Read-authority contract and consumer compatibility

**Status:** Ready
**Priority:** PF — Platform consumer read-authority extension
**Batch:** PF-B11
**Objective:** Freeze the versioned contract that makes Platform-backed local
consumer projections authoritative without introducing synchronous
per-request Platform reads.

## Authority map

- `docs/architecture/platform-ownership-and-boundaries.md` — canonical shared
  authority and product boundary.
- `docs/contracts/platform-contract-v1.md` — existing v1 identity, access and
  assertion authority; this Slice extends it with a separate read-authority
  contract and does not reinterpret its terminal assertion meaning.
- `docs/contracts/platform-events-v1.md` and
  `docs/contracts/platform-projection-v1.md` — event envelope and local
  projection-state rules.
- `docs/architecture/events-and-projection-contract.md` — ordered,
  idempotent, explicitly stale/gap/blocked projection behavior.
- `docs/technical/coding-standards.md` and the local validation contract —
  deep seams, server-side security and evidence classification.

## Current-state evidence

Platform `main` `e037873e9dfdfa22ce347b5ad371f47900f348e9` has canonical stores,
the v1 event envelope, an outbox and consumer inbox/checkpoint support. The
current contracts explicitly say that local projection support does not make a
projection authoritative and that no authority cutover is included. The
CTRL and IMS PD-D5-S01 plans are therefore correctly gated.

## Required contract output

Create `docs/contracts/platform-read-authority-v1.md` and its companion
`docs/contracts/platform-read-authority-compatibility-v1.md`. The contracts
must define:

- `platform.read-authority.v1` version negotiation and response headers;
- the exact shared entity and relationship allow-list, including active and
  archived/revoked state, canonical opaque IDs, role scope, source event ID,
  source sequence and source schema version;
- immutable bootstrap snapshot identity, checksum, expiry, deterministic
  record ordering, page limits and completion marker;
- opaque global change cursor plus per-aggregate sequence rules;
- `current`, `stale`, `gap`, `blocked`, `unavailable` and `resync_required`
  semantics, including the fail-closed rule;
- bounded endpoint/resource limits and non-sensitive error classes;
- per-consumer product key/audience compatibility for `tockrctrl` and
  `tockrims` without product-role or billing claims;
- machine request-signature requirements, key rotation, timestamp/nonce
  replay handling and transport trust assumptions;
- bootstrap, incremental sync, retry, cursor expiry, resync and rollback
  behavior; and
- compatibility, migration, retention and first-supported-version policy.

## Ordered work

### WP01 - Contract delta

Compare the existing v1 assertion, event and projection contracts with the
required read-authority behavior and record every non-overlapping delta.

Route: kind=other; risk=H[API,AUTH,DATA,DEP,DOC]

### WP02 - Wire and compatibility contract

Define the wire schemas, cursor/snapshot invariants, error taxonomy,
machine-authentication protocol and CTRL/IMS compatibility matrix in the new
contract artifacts.

Route: kind=authorization; risk=H[AUTH,API,DEP,GOV,DOC]

### WP03 - Staff Engineer plan/design review

Run the Staff Engineer plan/design review against current Platform code and
both consumer seams; resolve depth, locality, security, consistency and
rollback findings before marking S01 terminal.

Route: kind=other; risk=H[AUTH,CONC,API,DEP,GOV,DOC]

## Risk profile

`AUTH DATA CONC OPS API DEP GOV DOC` are high. `HIST` is high where source
provenance or revocation state is represented. No product-domain or UI scope
is authorised.

## Acceptance criteria

- **S01-AC01:** the new contract is versioned separately from the assertion
  contract and names every allowed/forbidden field and relationship.
- **S01-AC02:** bootstrap and incremental paths cannot claim current authority
  for expired, incomplete, blocked, unavailable, conflicting or gapped data.
- **S01-AC03:** machine authentication is explicit, rotated, replay-resistant
  and independent of browser sessions and user assertions.
- **S01-AC04:** CTRL and IMS can implement the same protocol with only their
  consumer key/product configuration differing; no product-specific role or
  billing fact crosses the boundary.
- **S01-AC05:** the Staff Engineer review is **PASS TO IMPLEMENTATION** with
  no unresolved material contract, security, data-integrity or rollback
  decision.

## Evidence and validation intent

Use format, architecture and security registry profiles after the contract
files exist, plus focused contract serialization/negative fixtures. Review
evidence must independently verify the field allow-list, version mismatch,
forbidden-field, cursor-gap, stale-state and request-signature decisions.
`python scripts/validate_plan_routing.py plan/active/pf-b11-s01-read-authority-contract.md`
must pass before delivery routing.

### TestContext and preflight

```text
candidate=<exact S01 implementation candidate under review>
authority=PF-B11-S01-AC01..AC05
surface=API|AUTH|DEP|DOC
profile=format,architecture,security plus focused contract tests
command_source=validation_registry.py and validate.py
preflight=PASS required before interpreting any behavioral result
```

Focused selectors must be preflighted; zero matching tests is
`INVOCATION_FAIL`, not PASS. Context failures remain `BLOCKED / NOT RUN`
and must not trigger application-code repair.

## Dependencies and stop/go

PF-B10-S02 is terminal. CTRL and IMS PD-D5-S01 plans have been inspected at
their current published heads. Stop if a contract decision would require a
shared database, synchronous per-request dependency, guessed identity mapping,
product authority, billing authority, or an edit to a terminal PF plan.

## Completion

Move this plan to `plan/completed/` only after the contract artifacts,
independent review, independent acceptance and exact-candidate local evidence
are complete. S01 completion authorises S02; it does not ungate consumer
cutover by itself.
