# PF-B5-S02 — User assignment and effective product access

Status: Planned

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

1. Define UserProductAssignment lifecycle and Organisation ownership invariant.
### WP01 - Ordered work package

Route: kind=authorization; risk=H[AUTH,DATA,GOV]
2. Implement one effective-access evaluator at the shared boundary; products
### WP02 - Ordered work package
   receive only shared identity/scope.
Route: kind=authorization; risk=H[AUTH,API,CONC]
3. Test every predicate independently and in combination, including revocation
### WP03 - Ordered work package
   and Workspace mismatch.
Route: kind=authorization; risk=H[AUTH,HIST]

## Migration impact

No CTRL/IMS import. Future reconciliation must preserve source facts and map
assignments explicitly.

## Security impact

High: membership alone, entitlement alone and assignment alone must all deny.
No product role, billing value or raw token crosses the predicate.

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
