# PF-B4-S01 — Workspace authority

Status: **Implemented / terminal.**

## Objective

Implement Organisation-owned Workspace lifecycle and generic
WorkspaceMembership with canonical admin/member/viewer roles.

## Authority and current evidence

Authority is the Platform hierarchy and role contract. CTRL/IMS PA work proves
Workspace ownership/access is a shared structural concern while product Project
roles remain local.

## Affected files/packages

Workspace domain/store/SQLite capability files, membership commands, audit,
HTTP boundary and behavior tests.

## Ordered work

### WP01 - Ordered work package
1. Define Workspace ownership, lifecycle and membership invariants.
Route: kind=authorization; risk=H[AUTH,GOV,DATA]
2. Implement transactional create/archive/membership transitions and default
### WP02 - Ordered work package
   selection without copying product roles.
Route: kind=authorization; risk=H[AUTH,DATA,HIST,CONC]
3. Add cross-Organisation ownership and inactive-Workspace negative tests.
### WP03 - Ordered work package

Route: kind=authorization; risk=H[AUTH,API]

## Migration impact

Fresh/upgrade/reopen tests are required. No existing CTRL/IMS Workspace is
assumed to be the same Platform Workspace.

## Security impact

Workspace access is explicit and server-side; membership is subordinate to an
active Organisation membership and Workspace ownership.

## Acceptance criteria

Every Workspace belongs to exactly one Organisation, generic roles are exact,
inactive or cross-tenant Workspaces deny access, and historical membership
changes are preserved.

## Tests and evidence

Store/HTTP behavior matrix, migration suite, concurrent membership transition
tests and independent tester acceptance.

## Dependencies

PF-B3-S02.

## Stop/go conditions

Stop for a requirement to embed CTRL/IMS Project roles or to infer membership
from product capability tables.

## Rollback

Use explicit archive/reopen policy and compensating membership changes; retain
history and do not drop Workspace tables.

## Terminal evidence

Accepted implementation candidate: `0835ba67bd38460676740cdf058e56b17b1330c7`.

- Independent engineering review: **PASS** with no R1 findings after the
  active-Organisation parent-authority repair.
- Independent tester acceptance: **PASS** for opaque Workspace identity,
  fixed admin/member/viewer roles, Organisation ownership, membership
  lifecycle/history, cross-tenant and inactive-parent denial, audit continuity,
  HTTP CSRF/redaction/safe errors and deterministic default selection.
- Exact-candidate local validation: `full/local` **PASS**; format,
  architecture, security, migration, frontend, quality, unit, SQLite/HTTP
  integration and race profiles **PASS**.
- Schema v5 fresh, upgrade, close/reopen and divergent-ledger evidence:
  **PASS**. Container build profiles are **NOT_APPLICABLE** because no
  authorised Dockerfile exists in this foundation scope.
- Concurrent membership assignment and stale Workspace-membership after
  Organisation revocation are explicitly covered and fail closed.
- Workspace membership remains subordinate to active Platform Organisation
  membership; CTRL/IMS Project roles, records, routes and authority are absent.
- The retained Platform SQLite policy remains one connection until a separate
  authorised, measured upgrade.
