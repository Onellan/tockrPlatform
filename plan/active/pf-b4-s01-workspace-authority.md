# PF-B4-S01 — Workspace authority

Status: Planned

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
