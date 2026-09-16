# PF-B5-S02 — User assignment and effective product access

Status: **Implemented / terminal**

## Objective

Implement the deny-by-default effective access predicate combining active
membership, Organisation entitlement, UserProductAssignment and Workspace scope.

## Authority and current evidence

Authority is Platform contract v1. CTRL/IMS PD plans explicitly require
authenticated User → membership → entitlement → assignment → Workspace access
before product authorization.

## Affected files/packages

Assignment domain/store/SQLite files, central access evaluator, HTTP/launcher
seams, audit and contract tests.

## Ordered work

### WP01 - Ordered work package

1. Define UserProductAssignment lifecycle and Organisation ownership invariant.

Route: kind=authorization; risk=H[AUTH,DATA,GOV]

### WP02 - Ordered work package

2. Implement one effective-access evaluator at the shared boundary; products
   receive only shared identity/scope.

Route: kind=authorization; risk=H[AUTH,API,CONC]

### WP03 - Ordered work package

3. Test every predicate independently and in combination, including revocation
   and Workspace mismatch.

Route: kind=authorization; risk=H[AUTH,HIST]

## Migration impact

Migration 7 adds the Platform-owned, history-preserving user product assignment
record and current-assignment uniqueness. No CTRL/IMS import occurs.

## Security impact

High: membership alone, entitlement alone and assignment alone all deny. No
product role, billing value or raw token crosses the predicate.

## Acceptance criteria

Access succeeds only when all required active predicates hold; revocation is
immediate; products can perform their own authorization after handoff.

## Tests and evidence

Truth-table authorization suite, tenant/Workspace isolation, revocation and
concurrency checks, audit and contract serialization tests.

## Dependencies

PF-B4-S02 and PF-B5-S01.

## Stop/go conditions

Stop if any caller bypasses the evaluator or if product access is inferred from
billing, plan name or membership.

## Rollback

Fail closed and disable handoff for the affected Product; retain assignment and
audit history for forward correction.

## Terminal evidence

Accepted implementation candidate: `1fbdb1a5a82b3e397d166d2cc24516d4cbf03597`.

- Independent engineering review: **PASS**, with no R1 findings.
- Independent tester acceptance: **PASS** for assignment lifecycle and
  organisation ownership, the all-predicate evaluator, fail-closed denial,
  immediate revocation, history/audit, concurrency and HTTP protection.
- Exact-candidate local validation on the implementation candidate:
  `format`, `architecture`, `security`, `migration`, `quality`, `unit`,
  `integration` and `race`: **PASS**. The first repository-wide race attempt
  exposed a pre-existing timing-sensitive MFA test failure; a repeat on the
  unchanged exact candidate passed without implementation changes. S02-focused
  race coverage also passed.
- Container build profiles: **NOT_APPLICABLE** because no authorised
  Dockerfile exists.
- The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  code, data, product role, migration or authority cutover was changed.
- PF-B6 and all later Batches remain unimplemented.

Detailed evidence:
[`PF-B5-S02 reconciliation`](../../docs/implementation/audits/pf-b5-s02-product-access.md),
[`independent review`](../../docs/implementation/audits/pf-b5-s02-independent-review.md),
[`independent acceptance`](../../docs/implementation/audits/pf-b5-s02-independent-acceptance.md).
