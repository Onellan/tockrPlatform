# PF-B2-S02 — Sessions, MFA, recovery and revocation

Status: Planned

## Objective

Provide secure database-backed sessions and applicable MFA/recovery controls
with immediate revocation at the protected request boundary.

## Authority and current evidence

Authority is the security runtime contract and current CTRL/IMS session hot-path
evidence. Both products use credential-free projections and database-backed
revocation, with different scope details retained locally.

## Affected files/packages

`internal/auth/session*`, `internal/store/session*`, SQLite session schema and
maintenance, HTTP middleware, recovery/MFA seams, templates and tests.

## Ordered work

1. Define hashed opaque token persistence, expiry, revocation and bounded
### WP01 - Ordered work package
   cleanup contracts.
Route: kind=authorization; risk=H[AUTH,DATA,OPS]
2. Implement authenticated request projection and MFA/recovery flows without
### WP02 - Ordered work package
   putting credentials in the projection.
Route: kind=authorization; risk=H[AUTH,API]
3. Prove revocation, expiry, inactive-user and CSRF behavior on every protected
### WP03 - Ordered work package
   boundary.
Route: kind=authorization; risk=H[AUTH,CONC,HIST]

## Migration impact

Schema-bearing work requires fresh, upgrade and reopen tests. Existing CTRL/IMS
sessions are never copied or accepted as Platform sessions.

## Security impact

High: replay, fixation, revocation lag, recovery abuse and secret disclosure are
explicit threat cases. Rate limiting/backoff and safe recovery are required.

## Acceptance criteria

Protected requests fail immediately for revoked/expired/inactive sessions;
session projections contain only required scope; MFA/recovery behavior is
bounded, auditable and fail closed.

## Tests and evidence

Session hot-path matrix, concurrent revocation check, CSRF/cookie tests, fresh/
upgrade/reopen migration tests, race profile where applicable and security
review.

## Dependencies

PF-B2-S01 and PF-B1-S03.

## Stop/go conditions

Stop if revocation depends on cache-only state, if recovery can bypass User
activation, or if Platform scope is mixed with product roles.

## Rollback

Revoke/disable the new session issuance path while preserving existing audit and
session records; no cross-repository session cutover is permitted.
