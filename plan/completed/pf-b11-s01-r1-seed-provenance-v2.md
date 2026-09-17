# PF-B11-S01-R1 — Seed provenance contract correction

**Status:** Implemented / terminal
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

This terminal Slice defines and implements the superseding
`platform.read-authority.v2` contract without changing v1. V2 represents
migration-created rows with explicit `migration_seed` provenance bound to the
ordered migration identity and represents post-outbox mutations with committed
event provenance. This Slice changes contract authority only; PF-B11-S02 still
owns snapshot storage and materialization.

## Non-negotiable boundaries

- The terminal v1 contract is unchanged and remains event-provenance-only.
- No synthetic `evt_*`, sequence or Product-created event is generated for a
  migration seed.
- The terminal `platform-events-v1` payload allow-list is unchanged.
- Platform remains authoritative for shared identity, tenancy, Product
  catalogue, entitlement and assignment; CTRL and IMS retain product roles,
  billing and local authorization.
- Consumers receive migration version/name/checksum, not SQL statements or
  database internals.

## Delivered work

### WP01 - Versioned contract artifact

Added complete v2 and compatibility-v2 documents covering exact version/header
behavior, unchanged entity allow-list, event/seed provenance, snapshot/feed
compatibility, rollback and consumer obligations.

Route: kind=other; risk=H[API,DATA,HIST,AUTH,DEP,GOV,DOC]

### WP02 - Deep validation seam

Extended `internal/platform/readauthority` with typed provenance and
version-aware validation. V1 accepts event provenance only. V2 requires exactly
one kind; event records require `evt_*`, positive sequence and schema version;
migration seeds require positive migration version, bounded name and valid
SHA-256 checksum. Mixed, malformed and whitespace-only metadata fails closed.

Route: kind=other; risk=H[DATA,HIST,API,DEP]

### WP03 - Compatibility handoff and regression evidence

Updated Platform, CTRL and IMS plans to require v2 support, seed preservation,
v1 event-feed compatibility and exact published-SHA handoff. Added focused
tests for valid event/seed records, serialization exclusivity, fabricated and
mixed provenance, malformed checksums and v1 rejection of seed provenance.

Route: kind=authorization; risk=H[AUTH,DATA,HIST,DEP,GOV,DOC]

## Candidate-bound evidence

Accepted implementation candidate:
`25c2502298b030f77e38aa246822611f875ad57a`.

- R1-AC01 through R1-AC06: **PASS**;
- independent engineering review: **PASS**;
- independent tester acceptance: **PASS**;
- exact-candidate format, quality, architecture, security, unit,
  integration, migration and race validation: **PASS**.

The initial review iteration found whitespace-only migration metadata handling;
the repair and wire-shape regression tests are included in the accepted
candidate. No required evidence is blocked or not run.

## Completion and handoff

PF-B11-S02 is promoted to **Ready** and must implement durable snapshot/source
cursor behavior against `platform.read-authority.v2`. S03 and S04 remain
dependency-bound. This Slice did not implement snapshot persistence, feed
routes, CTRL/IMS runtime code, product-role changes, billing, shared storage or
authority cutover.
