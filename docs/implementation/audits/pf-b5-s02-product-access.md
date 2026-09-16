# PF-B5-S02 reconciliation evidence

Accepted implementation candidate: `1fbdb1a5a82b3e397d166d2cc24516d4cbf03597`

Plan: [`plan/completed/pf-b5-s02-product-access.md`](../../../plan/completed/pf-b5-s02-product-access.md)

| Acceptance surface | Evidence | Result |
| --- | --- | --- |
| Assignment lifecycle | Platform-owned active/revoked UserProductAssignment with current-row uniqueness and preserved history | PASS |
| Organisation ownership | Only system administrators and authorised Organisation owner/admin actors can assign, revoke or list; targets must be active Organisation members | PASS |
| Central effective access | One transaction-time predicate requires active User, Organisation, membership, Product, entitlement, assignment, Workspace and permitted Workspace scope | PASS |
| Fail-closed denial | Owner without assignment, assignment without entitlement, wrong Workspace, non-member, inactive assignment, inactive entitlement and inactive membership all deny | PASS |
| Immediate revocation | Assignment and entitlement revocation deny on the next proof; assignment regrant preserves prior history | PASS |
| Audit and concurrency | Grant/revoke audit events are recorded; concurrent current assignment creation yields one success and one duplicate outcome | PASS |
| HTTP boundary | Mutations require CSRF, read/list scope is Organisation-authorized, access responses expose only generic shared identity/scope and safe denial | PASS |
| Migration safety | Migration 7 fresh/open/upgrade/ledger validation passed; no CTRL/IMS records or mappings are imported | PASS |
| Independent engineering review | Read-only review of exact candidate `1fbdb1a` | PASS; no R1 findings |
| Independent tester acceptance | Separate acceptance run on exact candidate `1fbdb1a` | PASS |
| Exact-candidate local validation | `format`, `architecture`, `security`, `migration`, `quality`, `unit`, `integration` and `race` profiles | PASS |
| Container profiles | AMD64 and ARM64 container builds | NOT_APPLICABLE; no authorised Dockerfile exists |

The initial one-connection SQLite policy remains unchanged. No CTRL/IMS route,
code, data, product role, migration or authority cutover was changed. The
first repository-wide race attempt failed in an existing MFA timing-sensitive
test; the unchanged exact candidate was rerun and the complete race profile
passed. That transient attempt is retained as diagnostic context, not as a
passing result.
