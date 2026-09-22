# PF-B12-S01 independent engineering review

**Implementation candidate:** `d246968b0a18e14883106283092998574e9a7e10`
**Certification candidate:** `87872417600a750a6cad0a81d2106c0f56ce78e6`
**Decision:** **PASS**
**Authority:** PF-B12-S01 WP01–WP04 and B12-AC01–AC08

The Staff Engineer reviewed the exact certification candidate read-only against
the PF-B12 Batch plan, the completed Slice plan, the membership command
contract, Platform ownership and boundary authority, the CTRL and IMS command
seams, and the repository validation authority.

## Findings

- The command surface is limited to OrganisationMembership and generic
  WorkspaceMembership add, role-change and deactivate operations. Organisation
  and Workspace lifecycle, entitlement, assignment and product-role writes are
  outside the contract.
- Product service authentication and Platform-issued acting-user proof are
  verified independently. Live Platform scope, role, active-user and
  lifecycle checks remain in the canonical Platform writers.
- Idempotency replay, altered-key conflict, expected-version conflict and
  concurrent-version behavior are transactionally bounded. Membership, audit
  and allow-listed outbox writes commit together; reasons and credentials stay
  out of feed payloads and diagnostics.
- Browser session/CSRF routes remain intact. CTRL and IMS retain disabled
  command seams and local projections; no consumer writer replacement,
  synchronous per-request dependency or production cutover was enabled.
- The certification candidate contains only the acceptance fixture repair after
  the implementation candidate. It reduces race-profile fixture overhead and
  preserves the archived-scope assertion in the existing authorization test.

No blocking engineering finding remains.

## Gate evidence

Exact-candidate `format`, `architecture`, `security`, `quality`, `migration`,
`unit`, `integration` and repository-wide `race` profiles passed on
`87872417600a750a6cad0a81d2106c0f56ce78e6`. Focused Platform HTTP/SQLite
command tests passed, and the published CTRL `2dbf1e1db7a1e6a33cb3f32f43d5b366f60aa1ff`
and IMS `852b3d01f2722db12a53caa734080bd665a829ca` disabled command seams
passed their focused client tests.
