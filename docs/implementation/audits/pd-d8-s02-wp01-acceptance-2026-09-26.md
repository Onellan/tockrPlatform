# PD-D8-S02 WP-PD8S02-01 tester acceptance — 2026-09-26

## Exact candidates

- Platform 4b507a88f7912288674b9cf78f57713abfa9e88a
- CTRL efd46bacd7d0041ebc18e375b275c4786d2b14b3
- IMS 225b193ba2ef90158529a92e7bf60b58822e7bdc

## Acceptance matrix

| AC | Evidence | Result |
| --- | --- | --- |
| AC-D8S02-01A | Roles and statuses compared for every shared entity and membership scope. | PASS |
| AC-D8S02-01B | ID prefixes and opaque identity rules compared; no local-ID derivation is accepted. | PASS |
| AC-D8S02-01C | Assertion algorithm, claims, audience/product binding, lifetime, skew and replay semantics compared. | PASS |
| AC-D8S02-01D | Entitlement, assignment, organisation/workspace scope and local product-role rules compared. | PASS |
| AC-D8S02-01E | Membership command operations, expected-version, idempotency, HTTP and error mappings compared. | PASS |
| AC-D8S02-01F | Snapshot/feed states, cursor ordering, checksum, stale/gap/resync and outage meanings compared. | PASS |
| AC-D8S02-01G | Repaired CTRL/IMS focused tests and Platform contract tests passed on the exact candidate set. | PASS |
| AC-D8S02-01H | Structured comparison contains zero unresolved rows or waivers. | PASS |

## Tester verdict

PASS — the semantic comparison is complete and has zero unresolved exceptions. This acceptance closes WP-PD8S02-01 only; it does not authorize D8-S02 sole-writer, retirement or cutover work.

