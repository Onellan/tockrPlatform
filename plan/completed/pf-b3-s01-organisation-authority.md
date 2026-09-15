# PF-B3-S01 — Organisation authority

Status: **Implemented / terminal.**

## Objective

Implement Organisation lifecycle and canonical OrganisationMembership authority
with Platform roles and history-preserving transitions.

## Authority and current evidence

Authority is the ownership contract and role table. CTRL PA and IMS PA show
Organisation membership already exists in product contexts but must become
shared Platform authority without copying product roles.

## Affected files/packages

`internal/domain/organisation*`, `internal/store/organisation*`,
`internal/db/sqlite/organisation*`, HTTP admin commands, audit and tests.

## Ordered work

1. Define Organisation lifecycle and membership invariants, including one
### WP01 - Ordered work package
   owner/admin/member role vocabulary.
Route: kind=authorization; risk=H[AUTH,GOV,HIST]
2. Implement transactional membership mutations with audit and active-state
### WP02 - Ordered work package
   checks.
Route: kind=authorization; risk=H[AUTH,DATA,HIST,CONC]
3. Add scope-negative tests proving cross-Organisation reads/writes fail closed.
### WP03 - Ordered work package

Route: kind=authorization; risk=H[AUTH,API]

## Migration impact

Platform-only fresh/upgrade schema. No automatic import or fabricated owner,
actor, timestamp or reason.

## Security impact

Organisation admin authority is server-side and must not be inferred from UI,
product role or membership presence alone.

## Acceptance criteria

Organisation and membership lifecycle is auditable, roles are canonical,
inactive relationships deny access, and every mutation proves current authority
inside its transaction.

## Tests and evidence

Store/HTTP behavior matrix, denied cross-tenant cases, audit/history checks,
fresh/upgrade/reopen migrations and independent acceptance.

## Dependencies

PF-B2-S02 and PF-B1-S02.

## Stop/go conditions

Stop for ambiguous owner transfer/deactivation semantics or any need to import
historical membership facts without provenance.

## Rollback

Use forward membership correction/revocation; never rewrite committed history or
delete the Organisation database to undo a failed transition.

## Terminal evidence

Accepted implementation candidate: `b82155c4606102750a537f6a5bc39be05939ed9e`.

- Independent engineering review: **PASS** with no R1 findings after the
  unscoped test-helper locality repair.
- Independent tester acceptance: **PASS** for Organisation lifecycle and audit,
  canonical owner/admin/member roles, inactive and cross-Organisation denial,
  transaction-bound authority checks and history-preserving transitions.
- Exact-candidate local validation: `full/local` **PASS**; format, architecture,
  security, migration, frontend, quality, unit, SQLite/HTTP integration and
  race profiles **PASS**.
- Fresh, upgrade, close/reopen and divergent-ledger migration evidence: **PASS**.
- Container build profiles are **NOT_APPLICABLE** because no Dockerfile is in
  this Slice's authorised scope.
- Owner transfer is deliberately not inferred: direct owner role mutation and
  owner membership deactivation are denied until a future authorised policy
  defines transfer semantics. Organisation archival is an explicit owner action
  that retires active memberships with audit history.
- No CTRL/IMS code, data, migration, product role or authority cutover was
  changed.
