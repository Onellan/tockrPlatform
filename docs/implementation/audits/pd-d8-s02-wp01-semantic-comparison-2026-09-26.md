# PD-D8-S02 WP-PD8S02-01 — Cross-repository semantic comparison

**Recorded:** 2026-09-26  
**Decision:** PASS — zero unresolved semantic exceptions  
**Scope:** Platform 4b507a88f7912288674b9cf78f57713abfa9e88a; CTRL efd46bacd7d0041ebc18e375b275c4786d2b14b3; IMS 225b193ba2ef90158529a92e7bf60b58822e7bdc

The CTRL and IMS SHAs are the repaired candidates. Each repository was clean, at origin/main, and ancestry-valid when this record was created.

## Authority baseline

- Platform is authoritative for User, Authentication, Organisation, OrganisationMembership, Workspace, WorkspaceMembership, Product, OrganisationProductEntitlement and UserProductAssignment.
- CTRL owns its project, control, WBS, timesheet, progress, forecasting, reporting and commercial facts.
- IMS owns its framework, governance, project, discipline, programme, assurance, document and provider/integration facts.
- CTRL and IMS consume local projections and bounded commands. Ordinary product requests do not synchronously call Platform or access the Platform database.
- Product-specific roles, capabilities, project access and billing decisions remain local to the product.

## Semantic comparison

| ID | Meaning compared | Platform meaning | CTRL implementation | IMS implementation | Resolution |
| --- | --- | --- | --- | --- | --- |
| SEM-01 | Organisation roles | owner, admin, member; owner mutation is separately protected. | Same three Platform roles; product roles are local. | Same three Platform roles; IMS roles are local. | MATCH |
| SEM-02 | Workspace roles | admin, member, viewer. | Same values and command validation. | Same values and command validation. | MATCH |
| SEM-03 | Record statuses | User/organisation/workspace: active or archived; memberships and entitlement/assignment: active or revoked; product: active or retired. | Same v2 record validation and projection meanings. | Same v2 record validation and projection meanings. | MATCH |
| SEM-04 | Opaque IDs | Type-bound prefixes usr_, org_, omem_, wsp_, wmem_, product., ent_, upa_; IDs are never derived from product-local IDs. | Same validation and explicit Platform identity mappings. | Same validation and explicit Platform identity mappings. | MATCH |
| SEM-05 | Assertion claims | Ed25519 TockrPlatformAssertion v1 contains issuer, audience, Platform user/org/workspace IDs, issued/expiry times, assertion ID and version; no product role or entitlement. | Strict signature, issuer, audience, typed ID, lifetime, skew and replay verification. | Same verification rules. | MATCH |
| SEM-06 | Assertion audience | tockrctrl maps to product.tockrctrl; tockrims maps to product.tockrims. | Adapter requires tockrctrl. | Adapter requires tockrims. | RESOLVED: intentional consumer specialization |
| SEM-07 | Shared access predicate | Active Platform product, organisation entitlement, user assignment and valid organisation/workspace scope are required; product roles remain local. | Local projection evaluates the same facts and freshness policy; no synchronous Platform dependency. | Same. | MATCH |
| SEM-08 | Projection freshness | current is authoritative; stale data supports only bounded degraded reads; commands require the fresh bound; expired/never-synced/non-current state blocks. | Fresh command bound 2 minutes; degraded read bound 15 minutes; non-current blocks. | Same bounds and decisions. | MATCH |
| SEM-09 | Membership command operations | platform.membership-command.v1; organisation/workspace scopes; add, change_role, deactivate; add expected version 0, other mutations positive. | Exact contract with actor assertion and idempotency key. | Exact contract with actor assertion and idempotency key. | MATCH |
| SEM-10 | Command errors | version_mismatch 406, unauthenticated 401, forbidden 403, invalid_request 400, conflict/stale 409, limit_exceeded 429, unavailable 503; replay mismatch is conflict with no mutation. | Every code maps to a stable local failure; no mutation inferred on failure. | Same. | MATCH |
| SEM-11 | Feed states | current, stale, gap, blocked, unavailable, resync_required; only current applies ordered changes; gap, stale, expiry or checksum failure requires bootstrap. | Ordered cursor and aggregate sequence checks; all resync-class errors enter resync state. | Same checks and transitions. | RESOLVED: cursor expiry and incomplete snapshot are explicit |
| SEM-12 | Read-authority errors | Includes version, authentication, access, request/limit, stale/gap/blocked/unavailable, resync, cursor/snapshot expiry, incomplete snapshot, checksum and conflict. | Explicit mappings for invalid request, rate limit, cursor expiry and incomplete snapshot; HTTP 406/400 fallbacks corrected. | Same explicit mappings. | RESOLVED: previous lossy mapping repaired |
| SEM-13 | Snapshot/feed integrity | v2 snapshots immutable and complete; checksum is lowercase SHA-256 of canonical records; cursors opaque; pages bounded; changes ordered and gap-checked. | Validates v2 metadata, records, checksums, cursors and limits. | Same validation. | RESOLVED: CTRL now rejects uppercase digest input |
| SEM-14 | Version/provenance boundary | Source retains terminal v1 event-only validation and v2 event-or-migration-seed validation; v2 is active read-authority contract. | Adapter sends/accepts v2 and validates v2 records only. | Same. | RESOLVED: v1 remains source-side compatibility only |
| SEM-15 | Error handling and data safety | Error bodies bounded and secret-free; failed commands and rejected feed pages fail closed without changing product data. | Strict JSON, bounded bodies, transaction boundaries and no product writes on rejected authority input. | Same. | MATCH |

## Repairs applied

1. CTRL canonical request validation now requires lowercase SHA-256 and no longer silently lowercases a different signed value.
2. CTRL and IMS expose and map cursor_expired and snapshot_incomplete, and map invalid_request and limit_exceeded; HTTP 406/400 fallbacks are consistent.
3. CTRL and IMS synchronizers classify cursor expiry and incomplete snapshots as resync_required.
4. Focused tests cover the repaired digest and error mappings.

## Evidence

- Platform: go test ./internal/platform/readauthority ./internal/platform/membershipcommand ./internal/platform/assertion ./internal/platform/http — PASS.
- CTRL: go test ./internal/platform/readauthority ./internal/platform/client ./internal/platform/sync — PASS.
- IMS: go test ./internal/platform/readauthority ./internal/platform/client ./internal/platform/sync — PASS.
- No unresolved rows, waivers or semantic exceptions remain.
- This work package does not authorize sole-writer proof, compatibility retirement or production cutover; those remain the ordered D8-S02 work packages.


