# PF-B4-S02 reconciliation evidence

Candidate: `9105b7debbb3aa857a1472373bb410676976900e`

Plan: [`plan/completed/pf-b4-s02-workspace-access.md`](../../../plan/completed/pf-b4-s02-workspace-access.md)

| Acceptance surface | Evidence | Result |
| --- | --- | --- |
| Narrow active-scope proof | `WorkspaceScope` store seam joins active User, Organisation, membership and Workspace state | PASS |
| Read/admin authorization matrix | Owner/admin, Workspace admin, member/viewer and system-admin tests | PASS |
| Protected caller coverage | Read/admin middleware wraps every `/api/workspaces/{workspaceID}` route; writers recheck inside transactions | PASS |
| Tampered and cross-tenant identifiers | Scope guard and HTTP safe-not-found tests | PASS |
| Revocation and inactive state | Inactive User, revoked Organisation membership, archived Workspace and concurrent revocation tests | PASS |
| Fail-before-mutation behavior | Revoked member archive attempt returns safe not-found and leaves Workspace active | PASS |
| Session hot path and boundary | Existing authenticated session middleware plus Workspace scope middleware; no product-role lookup | PASS |
| Independent engineering review | Read-only review of exact candidate `9105b7de` | PASS; no R1 findings |
| Independent tester acceptance | Separate acceptance run on exact candidate `9105b7de` | PASS |
| Exact-candidate local profile | `python scripts/validate.py run full/local` | PASS; container builds NOT_APPLICABLE without Dockerfile |
| Repository audit and boundary checks | `python scripts/audit_codebase.py`, architecture/security/quality/migration profiles and diff checks | PASS |

No CTRL/IMS route, code, data, product role or authority cutover was changed.
No shared database or migration beyond PF-B4-S01 was introduced.
