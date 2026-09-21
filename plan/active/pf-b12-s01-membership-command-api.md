# PF-B12-S01 — Versioned shared membership command API

**Status:** Ready
**Priority:** PF — shared-authority command extension
**Batch:** PF-B12
**Dependencies:** PF-B11-S04 terminal; CTRL and IMS PD-D5-S01 terminal; bounded
shared-writer authority decision recorded in the PF-B12 Batch plan.
**Baseline:** Platform `main` `81752c66b5677b4e964332b65a3bc1247b6124ff`;
CTRL `main` `89d430cb9c8768fa0da675b379a9ce542636935a`; IMS `main`
`7b19d0830a7dd8d566b895457bc0ec4fe788e8bf`.

## Objective

Publish and implement the Platform service command contract required by CTRL
and IMS PD-D5-S02 for canonical OrganisationMembership and generic
WorkspaceMembership mutations. Keep Platform authority centralized and keep
consumer projections local.

## Authority map

- PF-B12 explicit scope decision and B12-AC01..AC08 in
  [`../pf-b12-shared-membership-commands.md`](../pf-b12-shared-membership-commands.md).
- Platform ownership and role rules in
  [`../../docs/architecture/platform-ownership-and-boundaries.md`](../../docs/architecture/platform-ownership-and-boundaries.md).
- Consumer identity and assertion boundary in
  [`../../docs/contracts/platform-contract-v1.md`](../../docs/contracts/platform-contract-v1.md).
- Existing machine request signing and membership event allow-list in
  [`../../docs/contracts/platform-read-authority-v2.md`](../../docs/contracts/platform-read-authority-v2.md)
  and [`../../docs/contracts/platform-events-v1.md`](../../docs/contracts/platform-events-v1.md).
- Consumer command target seams: CTRL/IMS `internal/platform/client/client.go`.

## Current state and affected seams

Platform's existing REST member writes are registered under the session-
protected router and use CSRF for browser mutations. Platform store methods
already enforce canonical roles and persist actor/reason/audit facts with
membership events. The read-authority transport already verifies per-consumer
signatures, key rotation and replay-resistant nonces.

The consumers' `Commands` interfaces are currently disabled. Their command
structs only carry target IDs, role and active state; they do not yet carry a
verifiable acting-user proof, reason, stable idempotency key or expected
membership version. The contract and adapters must close those gaps without
moving product roles or authorization into Platform.

Affected Platform seams: `internal/platform/http/server.go`,
`internal/platform/http/organisation_admin.go`, existing membership store
commands, `internal/platform/auth/signed_request`, the event outbox, and a new
`docs/contracts/platform-membership-command-v1.md` contract. Reuse existing
capability-local HTTP/auth/store boundaries; do not add a generic command bus.

## Ordered work packages

### WP01 — Freeze contract and cross-product semantics

Compare current Platform/CTRL/IMS role vocabularies, add/change/deactivate
rules, actor attribution, validation, reasons and failure behavior. Publish the
versioned request/response contract, including service and acting-user proofs,
scope binding, idempotency, expected version, redacted result, error taxonomy,
compatibility and rollback. Stop on any semantic conflict that cannot be
resolved from existing authority.

Route: kind=authorization; risk=H[AUTH,DATA,HIST,API,DOC]

### WP02 — Add authenticated Platform command handlers

Register separate machine endpoints without weakening the browser session/CSRF
routes. Verify service signature and nonce, verify the Platform-issued actor
proof, resolve the actor's current Platform membership and target scope, then
reuse existing canonical add/change/deactivate store methods. Never accept a
caller-supplied actor ID as proof.

Route: kind=authorization; risk=H[AUTH,DATA,API,OPS,DEP]

### WP03 — Make mutation retry and transaction behavior explicit

Persist or otherwise durably resolve idempotency keys, reject key/payload
mismatch and stale expected membership versions, and ensure membership, audit
and outbox event commit atomically. Map existing Platform errors to the
versioned bounded failure response; keep private reason and secret material out
of feed and logs.

Route: kind=authorization; risk=H[DATA,AUTH,HIST,CONC,OPS,API]

### WP04 — Prove both consumer contracts and preserve browser behavior

Add independent service/actor authorization matrices for Organisation and
Workspace operations, including replay, rotation, stale actor, denied scope,
last-owner/lifecycle constraints, retries, conflicts, feed convergence and
secret-free diagnostics. Prove the browser session/CSRF path retains its
existing behavior. Add fixtures matching the CTRL and IMS client seams.

Route: kind=other; risk=H[AUTH,DATA,HIST,API,DEP]

## Acceptance mapping

| Acceptance | Work | Independent evidence |
| --- | --- | --- |
| B12-AC01 scope and role semantics | WP01, WP04 | Contract review and operation/role matrix |
| B12-AC02 separate product and acting-user authentication | WP01–WP04 | Valid/invalid consumer signature, actor proof, replay, rotation and scope matrix |
| B12-AC03 canonical live authorization and no-state-change denial | WP02, WP04 | Cross-tenant, inactive actor, stale proof and insufficient-role HTTP/store tests |
| B12-AC04 retry and stale-version conflicts | WP01, WP03, WP04 | Same-key replay, altered-payload rejection and concurrent-version tests |
| B12-AC05 atomic audit/outbox and redaction | WP02–WP04 | Transaction rollback, event allow-list and secret/reason log assertions |
| B12-AC06 CTRL/IMS compatibility | WP01, WP04 | One exact compatibility fixture per consumer |
| B12-AC07 browser routes and no consumer cutover | WP02, WP04 | Existing session/CSRF regression and disabled-consumer-mode assertion |
| B12-AC08 exact candidate release | all | Independent review, independent tester and repository-native exact-candidate validation |

## Validation and evidence

Use `format`, `unit`, `integration`, `security`, `architecture`, `migration`
and `race` through `scripts/validate.py`, narrowed or expanded by the
independent tester from affected surfaces. Use focused HTTP/store contract
tests for each acceptance row. Do not treat invocation/environment failures as
product failures or as passing evidence.

```text
candidate=<exact PF-B12-S01 candidate>
authority=PF-B12-S01 and B12-AC01..AC08
surface=AUTH|DATA|HIST|CONC|API|OPS|DEP
profile=format,unit,integration,security,architecture,migration,race
command_source=scripts/validation_registry.py and scripts/validate.py
preflight=PASS before behavioral interpretation
```

## Dependencies and stop/go

Platform PF-B11-S04 and both product PD-D5-S01 slices are terminal. The bounded
shared membership writer decision is approved for this plan. D6-S01 may
continue independently before consumer writer replacement. Do not enable the
consumer adapters or change production writer mode in this slice. Production
activation must wait for the existing PD-D7-S01 reconciliation and PD-D7-S02
cutover/rollback gates in each consumer.

## Completion

After B12-AC01..AC08 pass, complete independent review and acceptance, run the
selected exact-candidate local validation, publish the contract and endpoints,
and record the exact Platform candidate. Then promote CTRL/IMS D5-S02 from
Sequenced to Ready against that published endpoint version.
