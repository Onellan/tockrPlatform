# PF-B11-S01-R1 — Seed provenance contract correction

**Status:** Ready
**Priority:** PF — Platform consumer read-authority extension
**Batch:** PF-B11
**Supersedes:** The unresolved provenance decision recorded at PF-B11-S02;
this is a forward correction and does not rewrite terminal PF-B11-S01.
**Depends on:** PF-B11-S01 terminal; explicit authority to resolve the
fresh-install Product provenance conflict

## Authority and objective

The frozen `platform.read-authority.v1` contract requires event provenance for
every record, but Platform migration 6 creates the active
`product.tockrctrl` and `product.tockrims` rows before migration 8 creates the
outbox. No committed Product-created event exists and the v1 event allow-list
must remain unchanged.

Define and implement the superseding `platform.read-authority.v2` contract
without changing v1. V2 must represent migration-created rows with explicit
`migration_seed` provenance bound to the ordered migration identity and must
continue to represent post-outbox mutations with committed event provenance.
This slice changes contract authority only; PF-B11-S02 still owns snapshot
storage/materialization.

## Non-negotiable boundaries

- Never edit or reinterpret the terminal v1 contract.
- Never generate a synthetic `evt_*`, sequence or Product-created event for a
  migration seed.
- Never alter the terminal `platform-events-v1` payload allow-list.
- Keep Platform authoritative for shared identity, tenancy, Product catalogue,
  entitlement and assignment; CTRL and IMS retain product roles, billing and
  local authorization.
- Keep SQLite migration identity behind the Platform contract seam; consumers
  receive migration version/name/checksum, not SQL statements or database
  internals.

## Ordered work packages

### WP01 - Versioned contract artifact

Add complete v2 and compatibility-v2 documents. State unchanged v1/event
semantics, exact version/header behavior, record allow-list, seed/event
provenance, snapshot/feed compatibility, rollback and consumer obligations.

Route: kind=other; risk=H[API,DATA,HIST,AUTH,DEP,GOV,DOC]

### WP02 - Deep validation seam

Extend `internal/platform/readauthority` with typed provenance and
version-aware validation. V1 remains event-only. V2 requires exactly one
provenance kind; event records require `evt_*`, positive sequence and schema
version; migration seeds require positive migration version, bounded name and
valid SHA-256 checksum, with the other provenance fields absent.

Route: kind=other; risk=H[DATA,HIST,API,DEP]

### WP03 - Compatibility handoff and regression evidence

Update the Platform queue and the CTRL/IMS gated plans to require v2 support,
seed preservation, v1 event-feed compatibility and exact published SHA
handoff. Add focused tests for valid event/seed records and fabricated,
mixed, malformed and v1-incompatible provenance.

Route: kind=authorization; risk=H[AUTH,DATA,HIST,DEP,GOV,DOC]

## Acceptance criteria

- **R1-AC01:** v1 remains accepted only with event provenance and rejects
  migration-seed provenance; no terminal v1 document or event payload is
  rewritten.
- **R1-AC02:** v2 is an exact, independently named contract artifact and its
  record allow-list is unchanged except for the explicit provenance union.
- **R1-AC03:** v2 accepts a migration seed only with migration version/name and
  a valid checksum, rejects event/migration mixtures and rejects fabricated
  event identity for a seed.
- **R1-AC04:** v2 event provenance remains committed-event-only and the
  `platform-events-v1` allow-list is unchanged.
- **R1-AC05:** CTRL and IMS plans identify v2, preserve seed provenance locally,
  and remain gated until the complete Platform PF-B11 handoff is terminally
  published.
- **R1-AC06:** focused unit, format, quality, architecture and security
  validation passes on the exact candidate; no S02 runtime implementation is
  introduced by this correction.

## Evidence and validation intent

Use repository-resolved `unit`, `format`, `quality`, `architecture` and
`security` profiles. Establish candidate-bound TestContext before interpreting
behavioral evidence. Independent engineering review and independent tester
acceptance must inspect the exact final candidate separately. A context
failure is `BLOCKED / NOT RUN`, never a product pass or reason to mutate
application code.

```text
candidate=<exact R1 candidate>
authority=PF-B11-S01-R1-AC01..AC06
surface=API|DATA|HIST|AUTH|DEP|DOC
profiles=unit,format,quality,architecture,security
preflight=PASS required before behavioral interpretation
```

## Completion and handoff

After all gates pass, move this plan to `plan/completed/`, record the exact
candidate and evidence, update `plan/incomplete.md`, the PF-B11 index,
`docs/implementation/IMPLEMENTED.md` and related audits. Promote PF-B11-S02
to **Ready**, bind it to `platform.read-authority.v2`, and record the v2
contract requirement in CTRL and IMS without promoting their product slices.
