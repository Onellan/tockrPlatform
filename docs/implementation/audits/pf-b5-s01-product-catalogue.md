# PF-B5-S01 reconciliation evidence

Accepted implementation candidate: `9727f848c2e1c762ed8cc8fad6edfae59bf1de22`

Plan: [`plan/completed/pf-b5-s01-product-catalogue.md`](../../../plan/completed/pf-b5-s01-product-catalogue.md)

| Acceptance surface | Evidence | Result |
| --- | --- | --- |
| Initial catalogue | Migration 6 creates exactly `product.tockrctrl` and `product.tockrims` as independent active records | PASS |
| Stable product lifecycle | Product key is the immutable catalogue identifier; system-admin-only retirement records lifecycle state and time | PASS |
| Organisation entitlement authority | Owner/admin/system-authorized callers can grant or revoke an Organisation entitlement; members and outsiders fail closed | PASS |
| Entitlement and assignment separation | Entitlement changes do not create memberships, user assignments or an effective-access evaluator; PF-B5-S02 remains separate | PASS |
| Active-state checking | Entitlement listing derives active state from entitlement, Organisation and Product lifecycle | PASS |
| History and audit | Grant, revoke and regrant preserve rows and append Platform audit events with actor, product, entitlement, status and reason | PASS |
| Migration safety | Fresh, v5 upgrade, reopen and ledger-divergence evidence passed; initial migration does not fabricate actors or billing facts | PASS |
| HTTP boundary | CSRF, authorization, safe cross-tenant responses and redaction tests passed; no billing/payment/product-role fields are exposed | PASS |
| Independent engineering review | Read-only review of exact candidate `9727f848` | PASS; no R1 findings |
| Independent tester acceptance | Separate acceptance run on exact candidate `9727f848` | PASS |
| Exact-candidate local validation | `format`, `architecture`, `security`, `migration`, `quality`, `unit`, `integration` and `race` profiles | PASS |
| Container profiles | AMD64 and ARM64 container builds | NOT_APPLICABLE; no authorised Dockerfile exists |

The initial one-connection SQLite policy remains unchanged. No CTRL/IMS route,
code, data, product role, migration or authority cutover was changed.
