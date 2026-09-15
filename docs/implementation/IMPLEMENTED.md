# Implemented delivery ledger

The following Platform Foundation Slices are terminally recorded:

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
