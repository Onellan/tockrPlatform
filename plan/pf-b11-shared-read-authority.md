# PF-B11 — Shared read authority and consumer projection source

**Status:** Active at PF-B11-S03; PF-B1 through PF-B10 remain terminal historical
scope. PF-B11 is a separately authorised forward extension for the gated
CTRL/IMS PD-D5-S01 dependency.

**Priority:** PF — Platform consumer read-authority extension
**Authority:** Platform ownership and boundaries, the versioned Platform
contracts, the events/projection contract, the reconciliation contract, the
engineering workflow, and the Staff Engineer planning review recorded in
[`docs/implementation/audits/pf-b11-staff-engineer-plan-review.md`](../docs/implementation/audits/pf-b11-staff-engineer-plan-review.md).

## Objective

Publish and implement a versioned, bounded and fail-closed Platform
read-authority source that CTRL and IMS can consume into their own local
projections. The source must support an integrity-checked bootstrap snapshot,
an ordered incremental change feed, explicit projection freshness states and a
machine-authenticated transport without making Platform a synchronous
per-request dependency.

PF-B11 does not cut over CTRL or IMS, import product data, move product roles,
or create a shared product database. Those remain consumer-owned PD work.

## Current evidence and design decision

At Platform `main` `e037873e9dfdfa22ce347b5ad371f47900f348e9`, PF-B1 through
PF-B10 provide canonical shared records, signed assertions, a transactional
outbox, a bounded projection inbox/checkpoint seam and fixture-only
reconciliation. They do not provide a consumer read-authority wire contract,
an immutable bootstrap snapshot, a global resumable read cursor, or a
machine-authenticated consumer feed.

The Staff Engineer review therefore requires a new forward Batch rather than
editing terminal PF plans or treating the assertion contract as read
authority.

## Non-negotiable architecture decisions

1. Platform remains authoritative for User, Authentication, Organisation,
   OrganisationMembership, Workspace, WorkspaceMembership, Product,
   OrganisationProductEntitlement and UserProductAssignment.
2. CTRL and IMS retain their own local projection storage, product roles,
   product configuration, project/governance domains and local authorization.
3. Consumers bootstrap from an immutable bounded Platform snapshot and then
   consume an ordered change feed. They do not query Platform on every product
   request and never open the Platform SQLite database.
4. The terminal v1 read contract remains event-provenance-only. The current
   snapshot contract is the superseding `platform.read-authority.v2`, separate
   from the assertion contract `platform.v1`; v2 adds explicit migration-seed
   provenance without changing the v1 event feed. Unknown versions, invalid
   signatures, cursor gaps, expired snapshots and non-current source state fail
   closed.
5. The change cursor is opaque and monotonic at the Platform outbox boundary.
   Aggregate sequence remains part of the event contract; consumers must
   validate both the global cursor and per-aggregate ordering.
6. Machine transport authentication is separate from browser sessions and
   product assertions: per-consumer Ed25519 request signing, key IDs with
   overlap for rotation, bounded timestamp skew, durable nonce replay
   protection, constant-time verification and secret-free logs.
7. Snapshot and feed payloads contain only the allow-listed shared facts and
   provenance needed for projection. They exclude passwords, session tokens,
   private keys, product roles, billing/payment facts, document bytes and
   mutation reasons.

## Ordered Slice queue

| Order | Slice | Plan | State | Depends on |
| ---: | --- | --- | --- | --- |
| 1 | PF-B11-S01 — Read-authority contract and compatibility | [completed plan](completed/pf-b11-s01-read-authority-contract.md) | **Terminal** | PF-B10-S02 terminal; CTRL/IMS PD-D5 gate evidence |
| 2 | PF-B11-S01-R1 — Seed provenance contract correction | [completed plan](completed/pf-b11-s01-r1-seed-provenance-v2.md) | **Terminal** | PF-B11-S01 terminal; explicit provenance authority decision |
| 3 | PF-B11-S02 — Durable snapshot and source cursor | [completed plan](completed/pf-b11-s02-durable-snapshot.md) | **Terminal** | PF-B11-S01-R1 terminal; v2 provenance contract |
| 4 | PF-B11-S03 — Authenticated feed and resynchronisation API | [active plan](active/pf-b11-s03-feed-and-api.md) | **Ready** | PF-B11-S02 terminal |
| 5 | PF-B11-S04 — Security, operability and consumer-readiness certification | [active plan](active/pf-b11-s04-certification.md) | Planned | PF-B11-S01–S03 terminal; CTRL/IMS plan review |

Slices are strictly sequential. PF-B11-S01 freezes the terminal v1 contract;
PF-B11-S01-R1 supersedes the unresolved provenance decision with v2 and is
terminal. PF-B11-S02 is terminal and uses v2 migration-seed provenance for the
two migration 6 Product rows. Its accepted implementation candidate is
`065564e9db4be88dc556bb4b0fd0a88050c9487a`; independent review, tester
acceptance and exact-candidate local validation all passed. PF-B11-S03 is now
**Ready** and S04 remains unrun.

## Batch acceptance contract

- **B11-AC01:** `platform-read-authority.v2` defines canonical records,
  field allow-lists, relationship invariants, snapshot identity, cursor
  semantics, freshness states, bounded limits, error taxonomy, rollback /
  resynchronisation behavior and the event/migration-seed provenance union;
  terminal v1 event semantics remain unchanged.
- **B11-AC02:** a bootstrap snapshot is immutable, deterministic,
  checksum-bound, source-cursor-bound, expiry-bound and transactionally
  consistent across User, Organisation, Workspace, membership and access
  records.
- **B11-AC03:** the incremental feed is committed-source-only, bounded,
  resumable, duplicate-safe and explicit when the requested cursor requires
  resynchronisation.
- **B11-AC04:** only an authenticated consumer with a valid rotated signing
  key can access the integration surface; browser sessions and user assertions
  cannot substitute for machine authentication.
- **B11-AC05:** a source or projection state other than `current` cannot be
  represented as authoritative by the API or by consumer compatibility
  fixtures; no stale state can broaden access.
- **B11-AC06:** resource limits, rate limits, retention, cleanup, readiness,
  audit classification and secret-free operational diagnostics are explicit
  and tested.
- **B11-AC07:** independent review, independent tester acceptance,
  candidate-bound local validation, Batch certification and publication all
  bind to the same final candidate; CTRL and IMS receive the exact contract
  path/version and no cutover claim is made.

## Batch workflow and evidence

Lane 1 prepares the four Slice plans and validates routing. Lane 2 implements
one Slice at a time with RED→GREEN behavior evidence and narrow review. Lane 3
performs independent Slice acceptance, then full Batch review, acceptance and
exact-candidate certification. Use the repository registry through
`scripts/validate.py`; classify context failures separately from product
failures. Do not invoke GitHub Actions merely to validate implementation.

The Batch cannot close while the API contract is ambiguous, while transport
authentication is shared with browser sessions, while a snapshot is not
consistent, while a cursor can silently skip a gap, or while the consumer
handoff requires synchronous Platform reads.

## Publication boundary

PF-B11 publication makes the versioned Platform contract and implementation
available. It does not promote CTRL or IMS PD-D5-S01 automatically. After the
published candidate is independently verified, both consumer repositories
must update their gated plans with the exact Platform v2 contract version and
published SHA before either product promotes PD-D5-S01 to Ready.
