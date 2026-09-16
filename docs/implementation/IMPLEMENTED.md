# Implemented delivery ledger

The following Platform Foundation Slices are terminally recorded:

## PF-B5-S01 — Product catalogue and Organisation entitlements

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b5-s01-product-catalogue.md`](../../plan/completed/pf-b5-s01-product-catalogue.md)
- Accepted implementation candidate: `9727f848c2e1c762ed8cc8fad6edfae59bf1de22`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate format/architecture/security/migration/quality/unit,
  SQLite/HTTP integration and race validation all passed.
- Platform now owns the stable `product.tockrctrl` and `product.tockrims`
  catalogue records and auditable OrganisationProductEntitlement lifecycle.
  Product assignment and effective-access evaluation remain PF-B5-S02-owned.
- Container build profiles are **NOT_APPLICABLE** because no authorised
  Dockerfile exists. The initial one-connection SQLite policy is unchanged.
- No CTRL/IMS code, data, product role, migration or authority cutover was
  changed.

## PF-B5-S02 — User assignment and effective product access

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b5-s02-product-access.md`](../../plan/completed/pf-b5-s02-product-access.md)
- Accepted implementation candidate: `1fbdb1a5a82b3e397d166d2cc24516d4cbf03597`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate format/architecture/security/migration/quality/unit,
  SQLite/HTTP integration and race validation all passed.
- Platform now owns the history-preserving UserProductAssignment lifecycle and
  one deny-by-default effective-access predicate requiring active identity,
  Organisation membership, entitlement, assignment, Product, Workspace and
  permitted scope. Product-specific roles remain outside the proof.
- Migration 7 and the assignment HTTP seams preserve safe scope, CSRF and
  audit behavior. The initial one-connection SQLite policy is unchanged.
- Container build profiles are **NOT_APPLICABLE** because no authorised
  Dockerfile exists. No CTRL/IMS code, data, product role, migration or
  authority cutover was changed.

## PF-B5 — Product catalogue and product access

- Status: **Implemented / terminal Batch certification**
- Slices PF-B5-S01 and PF-B5-S02 are terminal in `plan/completed/` and were
  delivered strictly in dependency order.
- Accepted Batch candidate: `ffd051a093c07df66afc38e195b681d3bc2f444c`
- Independent Batch engineering review and independent Batch tester
  acceptance: **PASS**.
- Batch-local `full/local`, migration, assignment, effective-access, HTTP,
  audit/history, concurrency, plan-routing and boundary evidence: **PASS**;
  container builds **NOT_APPLICABLE** without an authorised Dockerfile.
- Platform owns the catalogue, Organisation entitlement, UserProductAssignment
  and shared effective-access predicate. Product-specific roles, billing and
  CTRL/IMS authority remain outside PF-B5.
- The initial one-connection SQLite policy remains unchanged. PF-B6-S01 is the
  next dependency-ready Slice; no later Batch is implemented here.

## PF-B1-S01 — Repository, standards, agents and validation foundation

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b1-s01-repository-foundation.md`](../../plan/completed/pf-b1-s01-repository-foundation.md)
- Accepted candidate: `d1737088b47e3180ac250f3f268c1d92e9719f17`
- Evidence: independent engineering review, independent tester acceptance,
  foundation audit, full/local validation, routing/contract tests and clean
  candidate-bound diff checks all passed.
- Runtime, migration, unit, integration, race and container profiles were
  `NOT_APPLICABLE` because their prerequisites are not introduced by this
  Slice.
- No CTRL/IMS code, data, migration or authority was changed.

## PF-B1-S02 — Platform ownership and shared contracts

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b1-s02-ownership-contracts.md`](../../plan/completed/pf-b1-s02-ownership-contracts.md)
- Accepted implementation candidate: `b8d8174ebb1ff659c3c3c7db5c420314b6f00b7c`
- Evidence: independent engineering review, independent tester acceptance,
  architecture/security/quality validation and contract assertions all passed.
- Runtime and data migration profiles were `NOT_APPLICABLE` because this Slice
  changes contracts and documentation only.
- No CTRL/IMS code, data, migration or authority was changed.

## PF-B1-S03 — Runtime, persistence and presentation foundation

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b1-s03-runtime-foundation.md`](../../plan/completed/pf-b1-s03-runtime-foundation.md)
- Accepted implementation candidate: `495d3e0278877d0f9c79fc8fb1f3e0ec65a7bf82`
- Initial SQLite policy: one connection, with WAL, serialized migration
  startup and a single-instance boundary. Any later pool-width upgrade is a
  separate authorized, measured change.
- Evidence: independent review, independent tester acceptance, foundation
  audit, full/local validation, routing/contract tests and exact-candidate
  diff checks all passed.
- Runtime/build profiles were `NOT_APPLICABLE` because no runtime module or
  Dockerfile is introduced by this planning Slice.
- No CTRL/IMS code, data, migration or authority was changed.

## PF-B2-S01 — User and authentication authority

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b2-s01-user-authentication.md`](../../plan/completed/pf-b2-s01-user-authentication.md)
- Accepted implementation candidate: `3d283961b8b6f88bd301712555587c2c556a857f`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate `full/local` validation, migration-ledger checks, secure
  cookie/CSRF/rate-limit tests, and one-connection SQLite evidence all passed.
- The Slice owns Platform identity/authentication only; no CTRL/IMS code, data,
  migration or authority cutover was changed.

## PF-B2-S02 — Sessions, MFA, recovery and revocation

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b2-s02-sessions-security.md`](../../plan/completed/pf-b2-s02-sessions-security.md)
- Accepted implementation candidate: `7a05b179419ebbe77dee18aaf1bace40d3f5fced`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate `full/local` validation, fresh/upgrade/reopen migration
  checks, race validation, protected-boundary tests, TOTP/recovery replay
  checks and bounded session cleanup all passed.
- No CTRL/IMS sessions, credentials, data, migration or authority was copied
  or cut over.

## PF-B2 — Identity and authentication

- Status: **Implemented / terminal Batch certification**
- Slices PF-B2-S01 and PF-B2-S02 are terminal in `plan/completed/` and were
  delivered strictly in dependency order.
- Accepted Batch candidate: `7a05b179419ebbe77dee18aaf1bace40d3f5fced`
- Independent Batch review and independent Batch tester acceptance: **PASS**.
- Batch-local full/local validation, migration/audit reconciliation and all 63
  Slice route signatures: **PASS**.
- Platform owns identity, authentication, sessions, MFA, recovery and
  revocation; CTRL/IMS authority and data remain outside this Batch.

## PF-B1 — Repository, standards and architecture foundation

- Status: **Implemented / terminal Batch certification**
- Slices PF-B1-S01, PF-B1-S02 and PF-B1-S03 are terminal in
  `plan/completed/`.
- Accepted Batch candidate: `495d3e0278877d0f9c79fc8fb1f3e0ec65a7bf82`
- Independent Batch review and independent Batch tester acceptance: **PASS**.
- Batch-local validation and all 63 Slice route signatures: **PASS**.
- No CTRL/IMS migration or authority cutover is claimed.

## PF-B3-S01 — Organisation authority

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b3-s01-organisation-authority.md`](../../plan/completed/pf-b3-s01-organisation-authority.md)
- Accepted implementation candidate: `b82155c4606102750a537f6a5bc39be05939ed9e`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate `full/local` validation, fresh/upgrade/reopen/divergence
  migration checks, transactional membership/audit tests and cross-Organisation
  denial tests all passed.
- Platform now owns Organisation lifecycle and canonical owner/admin/member
  membership history. Owner transfer semantics remain explicitly unresolved;
  direct owner mutation is denied rather than inferred.
- No CTRL/IMS code, data, migration or authority cutover was changed.

## PF-B3-S02 — Organisation administration seams

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b3-s02-organisation-administration.md`](../../plan/completed/pf-b3-s02-organisation-administration.md)
- Accepted implementation candidate: `5181e76de4b3feeb22b9bcc18b3929014915b286`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate `full/local` validation, v4 fresh/upgrade/reopen/divergence
  migration checks, HTTP authorization/CSRF/redaction/safe-error tests, audit
  continuity and Workspace entry-seam tests all passed.
- The Platform exposes narrow Organisation command/read seams only; system-role
  recognition is explicit, Workspace data remains PF-B4-owned and product roles
  remain absent.
- No CTRL/IMS route, code, data, migration or authority cutover was changed.

## PF-B3 — Organisation authority

- Status: **Implemented / terminal Batch certification**
- Slices PF-B3-S01 and PF-B3-S02 are terminal in `plan/completed/` and were
  delivered strictly in dependency order.
- Accepted Batch candidate: `5181e76de4b3feeb22b9bcc18b3929014915b286`
- Independent Batch review and independent Batch tester acceptance: **PASS**.
- Batch-local `full/local` validation, audit/reconciliation, migration and
  Platform boundary checks: **PASS**; container builds **NOT_APPLICABLE** without
  an authorised Dockerfile.
- PF-B4-S01 was the next dependency-ready Slice at this Batch certification
  point; it is now terminally recorded below. No CTRL/IMS authority cutover,
  product role or production-data import is claimed.

## PF-B4-S01 — Workspace authority

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b4-s01-workspace-authority.md`](../../plan/completed/pf-b4-s01-workspace-authority.md)
- Accepted implementation candidate: `0835ba67bd38460676740cdf058e56b17b1330c7`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate `full/local` validation, v5 fresh/upgrade/reopen/divergence
  migration checks, concurrent membership proof, HTTP authorization/CSRF/
  redaction/safe-error tests and Workspace audit/history assertions all passed.
- Platform now owns Organisation-owned Workspace lifecycle, generic
  admin/member/viewer membership, deterministic authorised default selection,
  audit continuity and active Organisation parent-scope enforcement.
- SQLite remains intentionally limited to one connection until a separate
  authorised, measured upgrade. Container builds are **NOT_APPLICABLE** without
  an authorised Dockerfile.
- No CTRL/IMS code, data, migration, product role or authority cutover was
  changed. PF-B4 is now terminally certified below.

## PF-B4-S02 — Workspace access and scope guard

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b4-s02-workspace-access.md`](../../plan/completed/pf-b4-s02-workspace-access.md)
- Accepted implementation candidate: `9105b7debbb3aa857a1472373bb410676976900e`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate `full/local` validation, active-scope truth-table tests,
  tampered/revoked/archived fail-closed tests, concurrent revocation proof,
  HTTP middleware coverage and race validation all passed.
- Platform now exposes one reusable User + Organisation + Workspace scope
  proof; protected Workspace reads and mutations use route middleware plus
  transaction-time writer rechecks. Product roles remain outside the proof.
- No CTRL/IMS code, data, product role or authority cutover was changed.

## PF-B4 — Workspace authority

- Status: **Implemented / terminal Batch certification**
- Slices PF-B4-S01 and PF-B4-S02 are terminal in `plan/completed/` and were
  delivered strictly in dependency order.
- Accepted Batch candidate: `9105b7debbb3aa857a1472373bb410676976900e`
- Independent Batch review and independent Batch tester acceptance: **PASS**.
- Batch-local `full/local` validation, Workspace migration/history/audit,
  scope-guard, HTTP and race evidence: **PASS**; container builds
  **NOT_APPLICABLE** without an authorised Dockerfile.
- Platform retains one SQLite connection. PF-B5-S01 is the next
  dependency-ready Slice; no CTRL/IMS migration or authority cutover is
  claimed.
