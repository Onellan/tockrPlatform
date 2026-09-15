# PF-B4-S02 — Workspace access and scope guard

Status: **Implemented / terminal.**

## Objective

Make User + Organisation + Workspace authorization a reusable, deep Platform
seam for protected requests and future product handoff.

## Authority and current evidence

Authority is the hierarchy/access contract and current CTRL/IMS session hot-path
evidence. The products prove active-scope guards are preferable to scattered
handler checks.

## Affected files/packages

`internal/store/access_scope*`, auth middleware/session projection, SQLite
scope queries/triggers where justified, HTTP routes and authorization tests.

## Ordered work

1. Define the narrow scope proof contract and required active-state joins.
### WP01 - Ordered work package

Route: kind=authorization; risk=H[AUTH,API,DATA]
2. Apply the guard to all Platform protected reads/writes and Workspace switch
### WP02 - Ordered work package
   paths.
Route: kind=authorization; risk=H[AUTH,CONC,API]
3. Prove tampered IDs, removed membership, archived Workspace and revocation
### WP03 - Ordered work package
   fail before mutation.
Route: kind=authorization; risk=H[AUTH,HIST]

## Migration impact

Schema changes require exact-prefix migration tests. No product request is
rewritten and no shared database is introduced.

## Security impact

High: this is the central IDOR and cross-tenant boundary. Fail closed on stale,
missing or contradictory scope.

## Acceptance criteria

One narrow guard proves active User, membership, Organisation ownership and
Workspace access; all protected callers use it; denied requests reveal no
protected resource information.

## Tests and evidence

Full authorization matrix, session hot-path tests, concurrent revocation/scope
switch tests, migration fresh/upgrade/reopen and race profile.

## Dependencies

PF-B2-S02, PF-B3-S02 and PF-B4-S01.

## Stop/go conditions

Stop if a caller can bypass the guard or if a product-specific role is required
to prove generic Workspace access.

## Rollback

Disable affected scope-switch/mutation paths while retaining the guard and
audit records; do not fall back to permissive authorization.

## Terminal evidence

Accepted implementation candidate: `9105b7debbb3aa857a1472373bb410676976900e`.

- Independent engineering review: **PASS** with no R1 findings.
- Independent tester acceptance: **PASS** for the active User,
  Organisation-membership, Workspace ownership/access and admin truth table;
  tampered IDs, inactive users, revoked parent membership and archived
  Workspaces fail closed before protected reads or mutations.
- Exact-candidate local validation: `full/local` **PASS**; format,
  architecture, security, migration, frontend, quality, unit, SQLite/HTTP
  integration and race profiles **PASS**.
- The reusable scope proof is exposed through the narrow Platform store seam and
  applied by read/admin HTTP middleware to every Workspace route. SQLite writer
  transactions re-prove the same active scope before mutation.
- No migration was required beyond the terminal PF-B4-S01 schema; no product
  request was rewritten and no shared database was introduced.
- Container build profiles are **NOT_APPLICABLE** because no authorised
  Dockerfile exists. No CTRL/IMS code, data, product role or authority cutover
  was changed.
