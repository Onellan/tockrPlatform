# PF-B6-S01 — Signed assertion issuance and verification

Status: **Implemented / terminal**

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

### WP02 - Ordered work package

2. Issue only after all active access predicates pass and include minimum
   `usr_`, `org_`, `wsp_` scope.

Route: kind=authorization; risk=H[AUTH,DATA]

### WP03 - Ordered work package

3. Verify signature, claims, expiry, audience and replay/ID semantics with
   negative fixtures.

Route: kind=authorization; risk=H[AUTH,CONC,OPS]

## Migration impact

No data migration. Key introduction uses secure startup configuration and an
explicit rotation runbook; existing product sessions are not upgraded silently.

## Security impact

High: private signing material is never logged or exposed, assertion lifetime
is bounded, unknown versions/keys/algorithms fail closed, and only shared
identity/scope claims cross the Platform boundary.

## Acceptance criteria

Assertions contain only approved shared claims, are short-lived/versioned,
verifiable by consumers and cannot be issued for inactive or unauthorized
scope.

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

## Terminal evidence

Accepted implementation candidate: `bf3b134e62155481cc98aad7b3613ccdc94129bd`.

Terminal closeout candidate: `cfe24a25e663147f07f8c1d5d5e8fe1e17246477`.

- Independent engineering review: **PASS**, with the public-key-only consumer
  verification repair reviewed as a material candidate change and no remaining
  R1 findings.
- Independent tester acceptance: **PASS** for the exact claim allow-list,
  signature and scope checks, expiry/future/replay denial, key rotation,
  public-key-only verification, access-gated HTTP issuance, CSRF and safe
  denial behavior.
- Exact-candidate local validation on the implementation candidate:
  `format`, `architecture`, `security`, `quality`, `unit`, `integration` and
  repository-wide `race`: **PASS**.
- Exact-candidate `full/local`, plan routing and read-only codebase audit on
  the terminal closeout candidate: **PASS**.
- Container build profiles: **NOT_APPLICABLE** because no authorised
  Dockerfile exists.
- The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  production code, data, role, migration or authority cutover was changed.
- PF-B6-S02 consumer compatibility and all later Batches remain unimplemented.

Detailed evidence:
[`PF-B6-S01 reconciliation`](../../docs/implementation/audits/pf-b6-s01-assertions.md),
[`independent review`](../../docs/implementation/audits/pf-b6-s01-independent-review.md),
[`independent acceptance`](../../docs/implementation/audits/pf-b6-s01-independent-acceptance.md).
