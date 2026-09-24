# PD-D7-S01-PF — Durable Platform production reconciliation/import boundary

**Priority:** PD prerequisite for CTRL/IMS D7-S01  
**Status:** Implementation candidate / WP-PD7PF-01 through WP-PD7PF-05 implemented; WP-PD7PF-06 remains gated on exact CTRL/IMS inventories and independent acceptance
**Planning baseline:** `d93aa6448fb953f2cd41766c3f70ccc0418f7660` (recheck `main` and the exact CTRL/IMS source candidates before implementation).  
**Owner:** TockrPlatform  
**Consumers:** `product.tockrctrl`, `product.tockrims`

## Objective

Replace PF-B9's deliberately fixture-only rehearsal with an explicitly authorised, durable production import boundary that can consume reviewed CTRL/IMS inventories, create or reconcile canonical Platform records, preserve source provenance and historical truth, resume safely, and compensate only records created by the verified import. This prerequisite unblocks consumer PD-D7-S01; it does not activate consumer authentication or membership writers.

## Authority map

- Platform ownership and boundaries: Platform remains the sole authority for User, authentication, Organisation, OrganisationMembership, Workspace, WorkspaceMembership, Product, OrganisationProductEntitlement and UserProductAssignment.
- PF-B9 reconciliation contract: source adapters provide canonical match keys; ambiguity, collision, missing provenance and unresolved relationships fail closed.
- `platform.read-authority.v2`: imported records become consumable only through the existing snapshot/feed and local projection path.
- `platform.membership-command.v1`: this plan may reuse canonical membership mutation/domain validation, but it must not broaden the command API or enable consumer writer mode.
- PD-D7-S01 CTRL/IMS plans: source inventories are read-only, exact-SHA bound, and no production import occurs until this plan is terminal.

## Current blocker evidence

PF-B9-S02 exposes only `FixtureOnlyScope`, `FixtureImporter`, an in-memory checkpoint and a manifest that contains entity, candidate Platform ID, match-key digest and source references. It has no SQLite persistence, production scope, payload needed to materialise records, product-access entity kinds, source snapshot receipt, operator execution API, or compensation against existing canonical records. The current PF-B9 contracts intentionally state that production import and authority cutover are outside scope.

These are separate implementation gaps, not one generic “importer” task:

1. Production manifest contract and payload are missing.
2. Product entitlement/assignment records are not representable by `EntityKind`.
3. Durable transaction/checkpoint/audit persistence is missing.
4. Existing-record match/reconcile and safe compensation semantics are missing.
5. Production operator authorization and key/receipt handling are missing.

## Affected seams and invariants

- Contract/types: `internal/platform/reconciliation/reconciliation.go`, `migration.go`, `internal/platform/membershipcommand/contract.go`, and new versioned production manifest types. Preserve `platform.reconciliation.inventory.v1` and `fixture-only` compatibility; add a new version/scope rather than changing terminal fixtures.
- Persistence: `internal/db/sqlite/store.go` migration ledger, canonical identity/organisation/workspace/membership/product tables, `platform_outbox`, audit tables and the existing one-connection SQLite policy.
- Domain/store: existing `organisation.go`, `workspace.go`, `product.go`, `identity.go` and transaction/audit helpers. Import code must call narrow domain seams or equivalent transaction-local helpers; it must not bypass authorization/history invariants with ad hoc SQL.
- Operations: new bounded CLI/API entrypoint must accept a signed manifest from a protected file/stream, return a non-secret receipt, expose status/checkpoint by manifest ID, and never print identity payloads or private keys.
- Projection handoff: canonical writes and outbox events commit atomically. A consumer bootstrap/feed must be able to reach `current`; unknown versions, gaps, stale state and conflicts remain blocked.

## Production manifest contract

Define `platform.reconciliation.import.v1` separately from the fixture manifest. It must bind:

- source report hash, exact CTRL/IMS source SHAs, manifest ID, schema/version and declared scope;
- operator approval ID, approval reason/time, execution authorization and key ID;
- deterministic records ordered by entity and canonical Platform ID;
- minimal canonical payload required to create/reconcile each entity: opaque IDs, parent IDs, user IDs, role, active state, display/name fields allowed by the canonical Platform model, product key/assignment state, and source provenance; never passwords, session secrets, assertion private keys or unapproved personal-data fields;
- explicit historical fields only when present in source evidence, with null preserved when unknown; no generated actor/time/reason may masquerade as history;
- source-to-canonical mapping and relationship dependencies so a manifest cannot apply a child before its verified parent/user;
- an import policy declaring create, reconcile-existing, conflict or blocked outcome for every record.

Extend the versioned reconciliation model with product, organisation-entitlement and user-assignment entity kinds or an explicitly versioned product-access import contract. Do not overload the terminal `platform.reconciliation.inventory.v1` five-entity schema.

## Ordered work packages

### WP-PD7PF-01 — Versioned production import contract

Specify canonical payload allowlists, entity dependencies, opaque ID rules, product-access records, source provenance, historical null semantics, deterministic ordering, manifest digest and signature domain. Add compatibility tests proving PF-B9 fixture manifests remain unchanged and production manifests cannot be accepted by fixture paths.

Route: kind=migration; risk=H[API,AUTH,DATA,GOV,HIST,DOC]

### WP-PD7PF-02 — Durable import ledger and checkpoint migration

Add versioned SQLite migrations for import runs, manifest records, source refs, approval/execution receipts, checkpoint integrity, per-record outcomes, conflict records, compensation records and operator audit. Enforce unique manifest identity, source snapshot binding, monotonic record order and one active execution per manifest. Include fresh/upgrade/reopen/divergence migration evidence.

Route: kind=migration; risk=H[DATA,CONC,HIST,OPS,DEPLOY]

### WP-PD7PF-03 — Transactional canonical apply and reconciliation

Implement dependency-ordered User → Organisation → Workspace → memberships → Product/entitlement/assignment apply using existing domain invariants. Reconcile an existing canonical record only when the source match key and immutable identity agree; classify drift, collision, missing dependency, inactive record and unsupported field as explicit outcomes. Commit canonical row, audit fact, outbox event and import outcome atomically; providers/network side effects remain outside the transaction.

Route: kind=migration; risk=H[AUTH,DATA,HIST,CONC,API]

### WP-PD7PF-04 — Resume, idempotency and compensating rollback

Resume after process/database failure from the durable checkpoint. Replaying a completed manifest is a no-op; the same manifest ID with a different digest, source SHA or payload fails closed. Rollback may compensate only records created by this manifest, must refuse to delete pre-existing or reconciled records, must preserve audit/outbox/import history, and must leave consumer re-bootstrap evidence.

Route: kind=migration; risk=H[AUTH,DATA,CONC,HIST,OPS]

### WP-PD7PF-05 — Protected operator boundary and receipts

Add a least-privilege command/API boundary with production-scope authorization, Ed25519 signature verification, key rotation/revocation, bounded body/record limits, replay protection, approval/execution separation, dry-run-before-apply, non-secret status output and signed result receipts. Refuse fixture scope, missing approval, stale source snapshot, untrusted source SHA and unsupported contract versions.

Route: kind=authorization; risk=H[AUTH,API,DATA,GOV,OPS]

### WP-PD7PF-06 — Consumer handoff and cutover gate

Run exact CTRL and IMS D7-S01 inventories through the production manifest in a disposable restored Platform database, prove snapshot/feed convergence and mapping receipts, then publish the operational runbook. Keep consumer modes local until both D7-S01 plans are terminal; this work package does not perform D7-S02 cutover.

Route: kind=migration; risk=H[AUTH,DATA,DEP,DEPLOY,OPS,GOV]

## Acceptance map

| AC | Required outcome | Work packages | Independent evidence |
| --- | --- | --- | --- |
| AC-PD7PF-01 | Production manifests are versioned, signed, deterministic and cannot enter fixture paths. | 01, 05 | Contract vectors, signature/replay/version negative matrix. |
| AC-PD7PF-02 | All shared-authority and product-access records needed by D7 are representable with bounded payload and truthful provenance. | 01, 03 | Entity/payload matrix and no-fabricated-history cases. |
| AC-PD7PF-03 | Apply is dependency ordered and atomic across canonical row, audit, outbox and ledger. | 02, 03 | Transaction rollback, outbox failure and fresh/upgrade/reopen migration tests. |
| AC-PD7PF-04 | Restart/replay is idempotent; conflicting manifests and unsafe reconciliation fail closed. | 02, 04 | Crash/resume, duplicate, digest-conflict and existing-record conflict tests. |
| AC-PD7PF-05 | Rollback compensates only manifest-created records and preserves historical evidence. | 04 | Exact-manifest rollback and pre-existing-record protection tests. |
| AC-PD7PF-06 | Operators receive a non-secret receipt and can prove source, approval, execution, checkpoint, outcomes and current projection state. | 05, 06 | Receipt verification, audit query and CTRL/IMS snapshot/feed convergence. |
| AC-PD7PF-07 | No consumer production authentication or membership writer mode is enabled by this plan. | 06 | Runtime config and cross-repository queue evidence. |

## Validation and publication

Use repository-resolved focused targets for reconciliation, import, migration, security and HTTP/operator boundaries, then `ci-core`, `ci-architecture`, `ci-security`, `ci-migration`, `ci-race`, `ci-quality` and applicable build profiles. Each result requires a `TestContext` with exact candidate, AC, surface, registry profile, command source and preflight PASS. Independent review, independent acceptance and exact-candidate validation are separate gates. Production rehearsal uses disposable/restorable data only; real CTRL/IMS execution requires the explicit D7 operator authorization recorded by the consumer plans.

## Implementation candidate

The candidate adds the versioned `platform.reconciliation.import.v1` production
manifest and Ed25519 approval/execution boundary, durable schema version 13
import-run/checkpoint/record ledgers, transactional dependency-ordered
canonical apply, checkpointed resume/idempotency, redacted status receipts,
and compensating rollback limited to records created by the manifest. The
terminal `platform.reconciliation.inventory.v1` fixture path remains unchanged
and is rejected by the production verifier. Product data, consumer writer mode
and authority cutover remain unchanged.

WP-PD7PF-06 is intentionally not claimed complete until exact matched CTRL/IMS
inventories are available, the disposable restored-database handoff converges
both consumers through snapshot/feed state, and independent review and
acceptance pass on the published candidate.

## Dependencies and stop/go

Dependencies: terminal PF-B9/PF-B11/PF-B12 contracts, matching CTRL/IMS D7-S01 inventory adapters and an owner-authorised production import decision. The plan may implement against fixtures and disposable restored databases before consumer data is authorized. Stop on payload overreach, unresolved identity semantics, missing source snapshot, conflicting existing canonical data, migration divergence, unverifiable backup/rollback, key/approval failure, or any request to activate consumer cutover.

## Mandatory completion requirement

Work is not complete while any blocker, bug, unresolved acceptance finding or required evidence row remains open. All implementation, review and acceptance findings must be repaired and rerun on the exact candidate; the accepted commit, plan status, queue and ledger must agree; and the clean accepted `main` must be pushed before this plan can be marked terminal.

## Completion

Move this plan to `plan/completed/` only after every AC passes, both consumer D7-S01 plans can proceed without a production-import blocker, independent gates pass, and the accepted Platform `main` is published.
