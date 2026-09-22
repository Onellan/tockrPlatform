# PF-B12-S01 independent tester acceptance

**Implementation candidate:** `d246968b0a18e14883106283092998574e9a7e10`
**Certification candidate:** `87872417600a750a6cad0a81d2106c0f56ce78e6`
**Decision:** **PASS**
**Authority:** PF-B12-S01 B12-AC01–B12-AC08

The tester independently executed the acceptance ledger against the exact
certification candidate. No implementation code was changed during this gate.

| Acceptance row | Evidence | Result |
| --- | --- | --- |
| B12-AC01 — bounded scope and role semantics | Versioned `platform.membership-command.v1` contract; out-of-scope operation rejection; Organisation and Workspace add/role-change/deactivate fixtures | PASS |
| B12-AC02 — separate product and acting-user authentication | Service signature, actor-proof signature binding, key/nonce checks, invalid proof and replay fixtures | PASS |
| B12-AC03 — live authorization and no-state-change denial | Inactive actor, stale assertion, cross-Organisation, cross-scope, insufficient-role and archived-scope HTTP tests | PASS |
| B12-AC04 — retry and stale-version conflicts | Same-key replay, altered payload conflict, stale version conflict and concurrent expected-version SQLite test | PASS |
| B12-AC05 — atomic audit/outbox and redaction | Outbox rollback, audit/result rollback, event allow-list and private reason redaction tests; bounded cleanup retains successful results for at least 24 hours | PASS |
| B12-AC06 — CTRL/IMS compatibility | CTRL Organisation and IMS Workspace exact compatibility fixtures; consumer heads `2dbf1e1…` and `852b3d0…` | PASS |
| B12-AC07 — browser behavior and no consumer cutover | Platform browser/CSRF regression plus CTRL/IMS disabled command seam tests with no transport call | PASS |
| B12-AC08 — exact release gate | Independent review PASS; exact-candidate format, unit, integration, migration, architecture, security, quality and race PASS | PASS |

## Evidence ledger

- **E01:** contract and scope matrix — PASS.
- **E02:** focused Platform HTTP/SQLite and consumer compatibility tests — PASS.
- **E03:** exact-candidate repository profiles on
  `87872417600a750a6cad0a81d2106c0f56ce78e6` — PASS, including repository
  race validation.
- **E04:** independent engineering review — PASS.

No required acceptance row is blocked. Consumer production writer activation
and PD-D7 cutover remain outside this Platform acceptance.
