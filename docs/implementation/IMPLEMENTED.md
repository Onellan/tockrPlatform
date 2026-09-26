# Implemented delivery ledger

## PD-D7-S01-PF — Durable Platform production reconciliation/import boundary

- Status: **Implemented / terminal**
- Completed plan: [`plan/completed/pd-d7-s01-platform-production-import.md`](../../plan/completed/pd-d7-s01-platform-production-import.md)
- Published Platform main: `223f03197147157bf90e0ee6cb7d8d25a1a7592d`
- Matched consumer main commits: CTRL `644dbb67ad6c685c343aa1ca51acc1985a9a9226`; IMS `69e2c7971ff12126acb564b1ce0a492dc592ab37`.
- Acceptance: [`pd-d7-s01-platform-import-acceptance-2026-09-26.md`](pd-d7-s01-platform-import-acceptance-2026-09-26.md)
- Platform now has the signed production import envelope, durable checkpointed apply/resume/idempotency, exact-manifest compensation, audit actor tombstone provenance, and operator receipts. WP-PD7PF-06 passed against the disposable restore point.
- CTRL and IMS each passed source-scoped snapshot/feed convergence and mapping finalization at `cur_22`; product-owned state and local authority modes remained unchanged. Consumer authentication and membership-writer cutover remain D7-S02 work.

## PF-B12-S01 — Versioned shared membership command API

- Status: **Implemented / terminal**
- Completed plan: [`plan/completed/pf-b12-s01-membership-command-api.md`](../../plan/completed/pf-b12-s01-membership-command-api.md)
- Implementation candidate: `d246968b0a18e14883106283092998574e9a7e10`
- Certification candidate: `87872417600a750a6cad0a81d2106c0f56ce78e6`
- Evidence: [`audits/pf-b12-s01-engineering-review.md`](audits/pf-b12-s01-engineering-review.md),
  [`audits/pf-b12-s01-tester-acceptance.md`](audits/pf-b12-s01-tester-acceptance.md)
  and [`audits/pf-b12-batch-certification.md`](audits/pf-b12-batch-certification.md).
- Platform now publishes `platform.membership-command.v1` for authenticated
  OrganisationMembership and generic WorkspaceMembership add, role-change and
  deactivate commands with live actor authorization, idempotency, expected
  versions, atomic audit/outbox writes and safe redaction.
- CTRL and IMS consumer command seams remain disabled/local-mode. D6-S01 may
  proceed before D5-S02; production writer activation remains gated by each
  product's D7 reconciliation and cutover plans.

## PF-B11-S01 — Read-authority contract and consumer compatibility

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b11-s01-read-authority-contract.md`](../../plan/completed/pf-b11-s01-read-authority-contract.md)
- Accepted implementation candidate: `b79b9321a06dd1c0e25381127dc61e861bae520d`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate format/architecture/security/unit and plan-routing checks all
  passed. The initial unsupported focused-validator invocation is retained as
  `INVOCATION_FAIL`; the repository-resolved unit profile passed and no
  required S01 evidence is blocked.
- Platform now publishes the separate `platform.read-authority.v1` contract,
  exact snapshot/feed/status envelopes, the CTRL/IMS compatibility matrix,
  fail-closed freshness states, bounded limits, machine-signing requirements
  and a capability-local validation seam. Snapshot persistence, feed routes
  and consumer cutover remain outside this Slice.
- No CTRL/IMS production code, data, migration, product role, billing fact,
  shared database or authority cutover was changed. PF-B11-S02 was initially
  **BLOCKED / NOT RUN** by the Product source-provenance conflict; the
  superseding S01-R1 correction below resolves that authority conflict and
  promotes S02 to Ready. No snapshot, feed or consumer cutover was started.

## PF-B11-S01-R1 — Seed provenance contract correction

- Status: **Implemented / terminal**
- Completed plan: [`plan/completed/pf-b11-s01-r1-seed-provenance-v2.md`](../../plan/completed/pf-b11-s01-r1-seed-provenance-v2.md)
- Accepted implementation candidate: `25c2502298b030f77e38aa246822611f875ad57a`
- Evidence: [`audits/pf-b11-s01-r1-implementation.md`](audits/pf-b11-s01-r1-implementation.md),
  [`audits/pf-b11-s01-r1-independent-review.md`](audits/pf-b11-s01-r1-independent-review.md),
  [`audits/pf-b11-s01-r1-independent-acceptance.md`](audits/pf-b11-s01-r1-independent-acceptance.md).
- The superseding `platform.read-authority.v2` contract adds explicit
  `migration_seed` version/name/checksum provenance for rows created before the
  outbox, while v1 and `platform-events-v1` remain unchanged.
- PF-B11-S02 was promoted to **Ready** by this correction; the correction did
  not implement snapshot persistence, feed routes or CTRL/IMS runtime behavior.

## PF-B11-S02 — Durable snapshot and source cursor

- Status: **Implemented / terminal**
- Completed plan: [`plan/completed/pf-b11-s02-durable-snapshot.md`](../../plan/completed/pf-b11-s02-durable-snapshot.md)
- Accepted implementation candidate: `065564e9db4be88dc556bb4b0fd0a88050c9487a`
- Evidence: [`audits/pf-b11-s02-engineering-review.md`](audits/pf-b11-s02-engineering-review.md),
  [`audits/pf-b11-s02-tester-acceptance.md`](audits/pf-b11-s02-tester-acceptance.md)
  and the retained initial-review repair record.
- Platform now provides a transactionally materialized, immutable and bounded
  `platform.read-authority.v2` bootstrap snapshot with an opaque source
  cursor, deterministic SHA-256 checksum, v2 event/migration-seed provenance,
  consumer-bound paging, expiry and bounded cleanup. Fresh/upgrade/reopen,
  corruption, omission, duplicate, source-mutation and race evidence passed.
- PF-B11 is now **terminally certified**. No consumer cutover, CTRL/IMS
  production code, product role, billing fact, shared database or authority
  transfer was changed.

## PF-B11-S03 — Authenticated feed and resynchronisation API

- Status: **Implemented / terminal**
- Completed plan: [`plan/completed/pf-b11-s03-feed-and-api.md`](../../plan/completed/pf-b11-s03-feed-and-api.md)
- Accepted implementation candidate: `bfc111ad1e7f8add6967e2dbb8b41f8ac2d192ea`
- Evidence: [`audits/pf-b11-s03-engineering-review.md`](audits/pf-b11-s03-engineering-review.md),
  [`audits/pf-b11-s03-tester-acceptance.md`](audits/pf-b11-s03-tester-acceptance.md)
  and the retained initial-review repair record.
- Platform now provides bounded v2 machine-authenticated snapshot, record,
  committed-change and status routes for `tockrctrl` and `tockrims`, with
  Ed25519 key overlap/retirement, durable nonce replay protection, bounded
  timestamp/body/page/response/rate controls, safe errors and explicit cursor
  resynchronisation. Snapshot provenance and terminal `platform-events-v1`
  payload semantics remain intact.
- All required local profiles and focused HTTP contract evidence passed on the
  exact candidate; no CTRL/IMS runtime, consumer cutover or shared database was
  implemented.

## PF-B11-S04 — Security, operability and consumer-readiness certification

- Status: **Implemented / terminal**
- Completed plan: [`plan/completed/pf-b11-s04-certification.md`](../../plan/completed/pf-b11-s04-certification.md)
- Certification candidate: `6da51a24b281549e5c8084f6c109bd860560154d`
- Published Platform `main` handoff: `737167fbbb2bb0c6746ec7d333ab9e6baf714c1f`
- Evidence: [`audits/pf-b11-s04-engineering-review.md`](audits/pf-b11-s04-engineering-review.md),
  [`audits/pf-b11-s04-tester-acceptance.md`](audits/pf-b11-s04-tester-acceptance.md)
  and [`audits/pf-b11-batch-certification.md`](audits/pf-b11-batch-certification.md).
- PF-B11 is terminally certified. All 27 PF Slices are reconciled; required
  validation passed; the unavailable Docker builds remain explicitly
  `BLOCKED / NOT RUN` and were not required because no container deployment
  surface changed. The Platform handoff makes CTRL and IMS eligible to
  re-evaluate PD-D5-S01 only; it does not implement their runtimes or
  authorize cutover.

The following Platform Foundation Slices are terminally recorded:

## PF-B7-S01 — Platform events and transactional outbox

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b7-s01-events.md`](../../plan/completed/pf-b7-s01-events.md)
- Accepted implementation candidate: `bde43056446327103f8e1241ce460aebe561bdcd`
- Terminal closeout candidate: `7ad1b8988fe81a0767b543fd09927a2d277a5e02`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate format/architecture/security/migration/frontend/quality/unit,
  SQLite/HTTP integration and extended repository-wide race validation all
  passed. The repository composite race child exceeded its fixed 300-second
  timeout and is retained as diagnostic `TIMEOUT`, not PASS evidence.
- Platform now emits versioned, bounded and redacted v1 outbox facts for
  committed shared identity, tenancy and product-access authority changes.
  Failed transactions leave no outbox fact; consumer ordering and duplicate
  detection use opaque event IDs and per-aggregate sequences.
- The initial one-connection SQLite policy remains unchanged. Container build
  profiles are **NOT_APPLICABLE** because no authorised Dockerfile exists. No
  CTRL/IMS production code, data, product role, migration or authority cutover
  was changed.

## PF-B7-S02 — Projection inbox and reconciliation support

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b7-s02-projections.md`](../../plan/completed/pf-b7-s02-projections.md)
- Accepted implementation candidate: `07c2b23ac35645b809fea3b1fc87042932f111b9`
- Terminal closeout candidate: `5d884cc3395e7ee6b2b7a11010f7bac3f435b8ba`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate format/architecture/security/migration/frontend/quality/unit,
  SQLite/HTTP integration and extended repository-wide race validation all
  passed. The repository composite race child exceeded its fixed 300-second
  timeout and is retained as diagnostic `TIMEOUT`, not PASS evidence.
- Platform now retains bounded consumer inbox identity, per-aggregate
  checkpoints and explicit current/stale/gap/blocked/unavailable states. Gap
  reconciliation is bounded and unknown source versions remain blocked for
  repair; projection state is never used as unconditional authorization.
- Migration 9 and the initial one-connection SQLite policy remain unchanged
  after acceptance. Container build profiles are **NOT_APPLICABLE** because no
  authorised Dockerfile exists. No CTRL/IMS production code, data, product
  role, migration or authority cutover was changed.

## PF-B7 — Events and local projection support

- Status: **Implemented / terminal Batch certification**
- Slices PF-B7-S01 and PF-B7-S02 are terminal in `plan/completed/` and were
  delivered strictly in dependency order.
- Accepted Batch candidate: `da66f502a170cc01c6cd1e8fe690d81f42bc5dd6`.
- Independent Batch engineering review and independent Batch tester
  acceptance: **PASS**.
- Batch-local format, architecture, security, migration, frontend, quality,
  unit, integration and extended repository-wide race evidence: **PASS**.
  The repository composite `full/local` race child exceeded its fixed
  300-second bound; the equivalent exact-candidate race command passed and the
  timeout is retained as diagnostic context. Container builds are
  **NOT_APPLICABLE** without an authorised Dockerfile.
- Platform now provides versioned event/outbox and bounded projection inbox/
  checkpoint support without making projections authoritative for access.
  The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  production code, data, product role, migration or authority cutover was
  changed.

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
- Accepted Batch candidate: `b6b79e5d8f92041e7335aac2a4f18b80f3d73e1a`
- Independent Batch engineering review and independent Batch tester
  acceptance: **PASS**.
- Batch-local `full/local`, migration, assignment, effective-access, HTTP,
  audit/history, concurrency, plan-routing and boundary evidence: **PASS**;
  container builds **NOT_APPLICABLE** without an authorised Dockerfile.
- Platform owns the catalogue, Organisation entitlement, UserProductAssignment
  and shared effective-access predicate. Product-specific roles, billing and
  CTRL/IMS authority remain outside PF-B5.
- The initial one-connection SQLite policy remains unchanged. PF-B7 is
  terminally certified above; PF-B9-S01 is now the next dependency-ready Slice
  and no later Batch is implemented here.

## PF-B6-S01 — Signed assertion issuance and verification

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b6-s01-assertions.md`](../../plan/completed/pf-b6-s01-assertions.md)
- Accepted implementation candidate: `bf3b134e62155481cc98aad7b3613ccdc94129bd`
- Terminal closeout candidate: `cfe24a25e663147f07f8c1d5d5e8fe1e17246477`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate format/architecture/security/quality/unit, SQLite/HTTP
  integration and repository-wide race validation all passed.
- Platform now issues a short-lived, versioned Ed25519 assertion only after
  the central effective-access proof succeeds. The strict shared claim set is
  limited to issuer, audience, `usr_`, `org_`, `wsp_`, issued/expiry times,
  assertion ID and version. Public-key-only consumer verification and key
  overlap are supported.
- Startup key configuration and the rotation runbook are explicit. The
  initial one-connection SQLite policy is unchanged.
- Container build profiles are **NOT_APPLICABLE** because no authorised
  Dockerfile exists. No CTRL/IMS production code, role, data, migration or
  authority cutover was changed.

## PF-B6-S02 — Consumer handoff and compatibility

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b6-s02-consumer-contract.md`](../../plan/completed/pf-b6-s02-consumer-contract.md)
- Accepted implementation candidate: `e9de6b100eafd19ad75a4b4c3046e0107cb93f62`
- Terminal closeout candidate: `9d87b50a1495776e49134249e6be009bc9dd55ca`
  (`full/local` PASS; container profiles **NOT_APPLICABLE**).
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate format/architecture/security/quality/unit/integration and
  repository-wide race validation all passed; uncached assertion and HTTP
  acceptance tests also passed.
- Platform now publishes the bounded v1 consumer compatibility matrix for
  `tockrctrl` and `tockrims`, with explicit product pairing and fail-closed
  unauthenticated, forbidden, stale, unavailable and version-mismatch classes.
  Product roles, billing, sessions, full entitlement detail and governance
  remain product-owned.
- The initial one-connection SQLite policy is unchanged. Container build
  profiles are **NOT_APPLICABLE** without an authorised Dockerfile. No CTRL/IMS
  production code, data, migration or authority cutover was changed.

## PF-B6 — Product assertion and consumer contract

- Status: **Implemented / terminal Batch certification**
- Slices PF-B6-S01 and PF-B6-S02 are terminal in `plan/completed/` and were
  delivered strictly in dependency order.
- Accepted Batch candidate: `eabcdf22c00d939fc07d5b1ea8eb69d903ab687a`.
- Independent Batch engineering review and independent Batch tester
  acceptance: **PASS**.
- Batch-local `full/local`, plan-routing, read-only codebase audit and boundary
  checks: **PASS**; container builds **NOT_APPLICABLE** without an authorised
  Dockerfile.
- Platform now has a versioned Ed25519 handoff assertion and an explicit
  consumer audience/product compatibility contract for CTRL and IMS. Product
  roles, billing, sessions, full entitlement detail and governance remain
  product-owned.
- The initial one-connection SQLite policy remains unchanged. PF-B7 is
  terminally certified above; PF-B9-S01 is now the next dependency-ready Slice
  and no later Batch or authority cutover is claimed.

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

## PF-B8-S01 — Platform layouts, selectors and launcher

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b8-s01-platform-shell.md`](../../plan/completed/pf-b8-s01-platform-shell.md)
- Accepted implementation candidate: `57b1669312d9336e7f5a0d0812e9135c38e75994`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate HTTP/presentation tests, frontend/architecture/security
  profiles, focused race validation and wide/narrow browser verification all
  passed. The `full/local` composite race child exceeded its fixed 300-second
  bound and is recorded as **TIMEOUT**, not PASS.
- Platform now provides the shared server-rendered shell,
  Organisation/Workspace selectors and access-gated product launcher. Product
  roles and operational screens remain outside Platform.
- The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  production code, data, migration, product role or authority cutover was
  changed. PF-B8 is now terminally certified and PF-B9-S01 is the next
  dependency-ready Slice.

## PF-B8-S02 — Platform administration UI

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b8-s02-administration-ui.md`](../../plan/completed/pf-b8-s02-administration-ui.md)
- Accepted implementation candidate: `13315d0fbb2c3b2163f9b34c4f8449de4cefb735`
- Evidence: independent engineering review, independent tester acceptance,
  exact-candidate Organisation/Workspace/System Admin HTTP tests, CSRF and
  fail-closed role checks, frontend/architecture/security profiles, focused
  race validation and wide/narrow browser verification all passed. The
  `full/local` composite race child exceeded its fixed 300-second bound and is
  recorded as **TIMEOUT**, not PASS.
- Platform now provides the initial server-rendered Organisation, Workspace
  and System Admin surfaces with recorded audit/history facts. Product roles,
  billing and CTRL/IMS operational screens remain outside Platform.
- The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  production code, data, migration, product role or authority cutover was
  changed.

## PF-B8 — Platform administration UI

- Status: **Implemented / terminal Batch certification**
- Slices PF-B8-S01 and PF-B8-S02 are terminal in `plan/completed/` and were
  delivered strictly in dependency order.
- Accepted Batch candidate: `13315d0fbb2c3b2163f9b34c4f8449de4cefb735`.
- Independent Batch engineering review and independent Batch tester
  acceptance: **PASS**.
- Batch-local named profiles, authorization/CSRF, audit/history, responsive
  browser and focused race evidence: **PASS**; the fixed `full/local` race
  child is retained as **TIMEOUT**, not converted to PASS. Container builds
  are **NOT_APPLICABLE** without an authorised Dockerfile.
- PF-B9 is now terminally recorded below. No CTRL/IMS authority cutover or
  product-role ownership is claimed.

## PF-B9-S01 — CTRL/IMS reconciliation inventory and mapping

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b9-s01-reconciliation.md`](../../plan/completed/pf-b9-s01-reconciliation.md)
- Accepted implementation candidate: `eafb9451572d248275f6eafe6174a547a4eceadb`
- Evidence: deterministic normalized inventory/proposal tests, collision and
  ambiguity blocking, source-linked relationship checks, redacted repeatable
  CLI output, independent review, independent tester acceptance and exact
  candidate `full/local` validation all passed. Container builds are
  **NOT_APPLICABLE** without an authorised Dockerfile.
- Platform now provides read-only fixture/adapter reconciliation proposals for
  Users, Organisations, Workspaces and memberships. Source/version/source-ID
  provenance is retained; no identity is accepted silently and unresolved
  mappings remain blocked.
- The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  production code, data, source connector, product role, import or authority
  cutover was changed.

## PF-B9-S02 — Dry-run/import and rollback tooling

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b9-s02-migration-tooling.md`](../../plan/completed/pf-b9-s02-migration-tooling.md)
- Accepted implementation candidate: `05b4cfb020dead9cc5cc1fcd22e8bb2cb671489d`
- Evidence: signed fixture-manifest approval and verification, deterministic
  fixture import, HMAC checkpoint integrity, pause/resume, idempotency,
  compensating rollback, audit retention, independent review, independent
  tester acceptance and exact-candidate `full/local` validation all passed.
  Container builds are **NOT_APPLICABLE** without an authorised Dockerfile.
- Platform now has a fixture-only rehearsal boundary for staged migration
  manifests. Blocked/ambiguous records cannot be imported, and production
  import, shadow mode and authority cutover remain outside PF.
- No Platform SQLite migration, CTRL/IMS source connector, production record,
  product role or authority cutover was added. PF-B9 is terminally certified;
  PF-B10-S02 is the next dependency-ready Slice.

## PF-B9 — CTRL/IMS reconciliation and migration tooling

- Status: **Implemented / terminal Batch certification**
- Slices PF-B9-S01 and PF-B9-S02 are terminal in `plan/completed/` and were
  delivered strictly in dependency order.
- Accepted Batch candidate: `9c073cba5ce495de9bc217696062485c9999ab62`.
- Independent Batch engineering review and independent Batch tester
  acceptance: **PASS**.
- Batch-local signed-manifest, fixture import/rollback, provenance, profile
  and race evidence: **PASS**; container builds are **NOT_APPLICABLE** without
  an authorised Dockerfile.
- PF-B10-S02 is now the next dependency-ready Slice. No production import,
  consumer-repository migration or authority cutover is claimed.

## PF-B10-S01 — Security and hardened runtime

- Status: **Implemented / terminal**
- Historical plan: [`plan/completed/pf-b10-s01-runtime-hardening.md`](../../plan/completed/pf-b10-s01-runtime-hardening.md)
- Accepted implementation candidate: `6cdfed1179d4f0dbc5266991ad6074741ef7dd75`
- Independent engineering review, independent tester acceptance and exact
  candidate `full/local` validation all passed. Format, architecture,
  security, migration, frontend, quality, unit, integration, repository-wide
  race, AMD64 and ARM64 container build profiles all passed.
- Platform now has strict secret-safe startup configuration, bounded HTTP
  resources, restrictive security headers, graceful shutdown, distinct safe
  `/healthz` and SQLite-backed `/readyz`, and hardened non-root container
  targets for Linux AMD64/ARM64. Compose applies read-only root storage,
  dropped capabilities, bounded `/tmp`, no-new-privileges and a persistent
  Platform data volume.
- The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  production code, data, connector, product role, production record or
  authority cutover was changed. PF-B10-S01-R1 and PF-B10-S02 are terminally
  recorded below; PF-B10 is certified.

## PF-B10-S01-R1 — Runtime asset packaging repair

- Status: **Implemented / terminal**
- Completed plan: [`plan/completed/pf-b10-s01-r1-runtime-assets.md`](../../plan/completed/pf-b10-s01-r1-runtime-assets.md)
- Accepted repair candidate: `2dbac87909b296e66da33f1e9f26d049bbbd1bd7`
- Independent engineering review, independent tester acceptance, exact
  `full/local` validation, AMD64/ARM64 builds and real-browser `/login`,
  `/healthz`, `/readyz`, CSS MIME and favicon checks all passed.
- The repair packages the existing source-owned static assets in the hardened
  image and adds a safe `204` favicon response. It introduces no new product
  authority, schema, migration, SQLite pool change or CTRL/IMS code/data.
- PF-B10-S02 was promoted only after this repair was terminally published and
  is certified below.

## PF-B10-S02 — Final Platform foundation certification

- Status: **Implemented / terminal**
- Completed plan: [`plan/completed/pf-b10-s02-final-certification.md`](../../plan/completed/pf-b10-s02-final-certification.md)
- Validated certification candidate: `26d6923dfc277e71a1253b110bd6f740ce3b475f`
- Independent engineering review, independent tester acceptance, final
  certification reconciliation and exact `full/local` validation all passed.
- All PF plan rows and prior Batch evidence reconcile through PF-B10. No later
  Slice was implemented in parallel, and no CTRL/IMS migration, production
  import, authority cutover or SQLite pool upgrade is authorized.

## PF-B10 — Security, runtime and final certification

- Status: **Implemented / terminal Batch certification**
- Batch certification: [`docs/implementation/audits/pf-b10-batch-certification.md`](audits/pf-b10-batch-certification.md)
- PF-B10-S01, PF-B10-S01-R1 and PF-B10-S02 are terminal and were delivered in
  strict dependency order with candidate-bound independent gates.
- The Platform Foundation programme is terminal at the authorised scope. Future
  runtime upgrades or cross-repository authority changes require a new explicit
  plan and independent acceptance.
