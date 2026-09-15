# PF-B6-S02 — Consumer handoff and compatibility

Status: Planned

## Objective

Publish and test the bounded CTRL/IMS consumer contract without changing either
product’s authentication or authority in PF.

## Authority and current evidence

Authority is the versioned assertion contract and the CTRL/IMS PD integration
plans. Consumer adapters are future product-owned work; Platform owns issuance
and contract compatibility.

## Affected files/packages

Platform assertion endpoint/contract docs, consumer fixtures, compatibility
matrix and integration test harness; no CTRL/IMS production code.

## Ordered work

1. Define consumer audience/product key/version negotiation and failure taxonomy.
### WP01 - Ordered work package

Route: kind=authorization; risk=H[AUTH,API,DEP]
2. Provide test fixtures for valid, expired, revoked, wrong-audience,
### WP02 - Ordered work package
   wrong-product and stale-scope assertions.
Route: kind=authorization; risk=H[AUTH,API]
3. Prove contract compatibility and publish handoff guidance without claiming
### WP03 - Ordered work package
   cutover.
Route: kind=other; risk=H[DOC,DEPLOY]

## Migration impact

None. Product-side migration and coexistence remain outside this Slice.

## Security impact

Consumer failures must distinguish unauthenticated, forbidden, stale,
unavailable and version mismatch without leaking scope.

## Acceptance criteria

CTRL and IMS can independently implement a consumer against the versioned
contract; no product role/billing claim is required; invalid claims fail closed.

## Tests and evidence

Contract serialization, compatibility and negative matrix; local integration
profile and independent review. No remote CI-only claim.

## Dependencies

PF-B6-S01 and PF-B5-S02.

## Stop/go conditions

Stop if consumer behavior requires a shared database, synchronous per-request
Platform dependency or product-specific authority in Platform.

## Rollback

Withdraw the contract version and retain the previous supported version only if
its security properties remain valid; no product cutover is rolled back here.
