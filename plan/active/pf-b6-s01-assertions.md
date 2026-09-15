# PF-B6-S01 — Signed assertion issuance and verification

Status: Planned

## Objective

Implement the short-lived, versioned signed Platform assertion for CTRL/IMS
handoff with minimum shared identity/scope claims.

## Authority and current evidence

Authority is `docs/contracts/platform-contract-v1.md` and the PF assertion
requirements. CTRL/IMS PD contracts require a Platform-signed assertion without
product roles or billing.

## Affected files/packages

`internal/platform/assertion`, key/config seams, access evaluator integration,
HTTP handoff boundary, contract fixtures and security tests.

## Ordered work

### WP01 - Ordered work package
1. Define issuer/audience/version/key-rotation and expiry contract.
Route: kind=authorization; risk=H[AUTH,API,DEP]
2. Issue only after all active access predicates pass and include minimum
### WP02 - Ordered work package
   `usr_`, `org_`, `wsp_` scope.
Route: kind=authorization; risk=H[AUTH,DATA]
3. Verify signature, claims, expiry, audience and replay/ID semantics with
### WP03 - Ordered work package
   negative fixtures.
Route: kind=authorization; risk=H[AUTH,CONC,OPS]

## Migration impact

No data migration. Key introduction requires secure configuration and rotation
runbook; existing product sessions are not upgraded silently.

## Security impact

High: protect private signing material, avoid secret logs, bound lifetime and
fail closed on unknown versions or invalid scope.

## Acceptance criteria

Assertions contain only approved shared claims, are short-lived/versioned,
verifiable by consumers and cannot be issued for inactive or unauthorized scope.

## Tests and evidence

Cryptographic contract tests, claim allow-list, expiry/replay/key-rotation
tests, secret scan and independent security review.

## Dependencies

PF-B5-S02.

## Stop/go conditions

Stop if a product asks for roles/billing/full entitlement data or if key
management cannot be made explicit and auditable.

## Rollback

Revoke the assertion key/version and disable handoff issuance; do not accept
older unsafe claims as a compatibility shortcut.
