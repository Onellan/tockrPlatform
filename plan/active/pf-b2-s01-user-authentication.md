# PF-B2-S01 — User and authentication authority

Status: Planned

## Objective

Implement Platform-owned User lifecycle and secure authentication through
capability-local domain, store and HTTP seams.

## Authority and current evidence

Authority is PF ownership, the security runtime contract and contract v1.
CTRL/IMS both provide current authentication/password/TOTP/session patterns;
their product-specific users are source records, not Platform data.

## Affected files/packages

`internal/domain/user*`, `internal/store/user*`, `internal/db/sqlite/user*`,
`internal/auth`, `internal/platform/http/auth*`, templates and focused tests.

## Ordered work

1. Define User lifecycle, opaque ID generation and credential storage contracts.
### WP01 - Ordered work package

Route: kind=authorization; risk=H[AUTH,DATA,HIST]
2. Implement login, logout, password verification and unknown-user fixed-cost
### WP02 - Ordered work package
   handling at the HTTP boundary.
Route: kind=authorization; risk=H[AUTH,OPS,API]
3. Add allowed/denied behavior tests and audit/security events without exposing
### WP03 - Ordered work package
   secrets.
Route: kind=authorization; risk=H[AUTH,DOC]

## Migration impact

Fresh Platform database only. No CTRL/IMS users are imported by this Slice.

## Security impact

High: secure password hashing, rate limiting/backoff, secure cookies, CSRF,
safe errors and secret-free logs are mandatory.

## Acceptance criteria

Users have stable `usr_` IDs, active/inactive lifecycle, secure sign-in/out,
server-side authorization and auditable security changes; invalid credentials
do not disclose account existence.

## Tests and evidence

Focused auth profile with valid TestContext; allowed/denied login matrix,
CSRF/cookie/header checks, rate-limit behavior, audit assertions and fresh DB
tests. Reviewer and tester remain independent.

## Dependencies

PF-B1-S03.

## Stop/go conditions

Stop if credential or session authority is ambiguous, if raw secrets enter
storage/logs, or if a product role is needed to authenticate.

## Rollback

Disable the new routes/configuration and restore the previous Platform schema
prefix; do not delete user history or modify CTRL/IMS auth.
