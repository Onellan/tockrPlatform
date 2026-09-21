# PF-B12 — Machine-authenticated shared membership commands

**Status:** Active forward extension, separately authorised by the 2026-09-21
request to unblock PD-D5-S02. PF-B1 through PF-B11 remain terminal at their
accepted scopes.
**Priority:** PF — shared-authority command extension
**Baseline:** Platform `main` `81752c66b5677b4e964332b65a3bc1247b6124ff`,
CTRL `main` `89d430cb9c8768fa0da675b379a9ce542636935a`, and IMS `main`
`7b19d0830a7dd8d566b895457bc0ec4fe788e8bf`.

## Objective

Provide the versioned Platform command API needed for CTRL and IMS to route
OrganisationMembership and generic WorkspaceMembership administration to the
canonical Platform writers. Platform remains the sole steady-state writer for
those shared facts. Product roles, capabilities, settings and product-domain
records remain in their owning product.

## Explicit scope authority

The user request authorises only the existing PD-D5-S02 scope: Platform as the
single steady-state writer for OrganisationMembership and generic
WorkspaceMembership changes consumed by CTRL and IMS. It does not authorise
Organisation or Workspace lifecycle writes, product entitlement or assignment
commands, billing/payment authority, product-role changes, migration/import,
authentication cutover, production writer cutover or compatibility retirement.

The command API must authenticate both the calling product and the acting
Platform user. Platform derives the actor from a verifiable Platform-issued
identity proof and checks the actor's current Platform authority for the target
scope. A product key, user ID or product-role claim supplied by a consumer is
not actor or permission proof.

## Current evidence

- PF-B11 is terminal and published at the baseline SHA above. Its signed
  machine transport and ordered `platform-events-v1` feed are available.
- Platform already has canonical add, role-change and deactivate store
  operations with actor, reason, audit and membership event writes.
- The current HTTP membership write routes require a Platform browser session
  and CSRF; they are not service command endpoints.
- Both consumers have disabled `Commands` seams, but their current command
  values lack actor proof, reason, request idempotency and expected version.
- CTRL and IMS D5-S02 plans have matching membership scope and acceptance
  intent. D6-S01 remains ordered first so command results can converge through
  the durable local projection feed before consumer writer replacement.

## Authority map

- [Platform ownership and boundaries](../docs/architecture/platform-ownership-and-boundaries.md): Platform membership roles and canonical writer.
- [Platform contract v1](../docs/contracts/platform-contract-v1.md): consumer and acting-user identity boundary.
- [Platform events v1](../docs/contracts/platform-events-v1.md): existing membership event allow-list and redaction.
- [Platform read-authority v2](../docs/contracts/platform-read-authority-v2.md): service signing, nonce replay protection and consumer identity.
- CTRL and IMS `plan/pd-d5-s02-platform-authority-writes.md`: approved membership-only consumer scope.

## Ordered work

1. Freeze the cross-product command semantics and publish a versioned command
   contract. Reconcile current Platform, CTRL and IMS role/lifecycle behavior;
   stop if a difference cannot be resolved from existing authority.
2. Add a narrow machine-to-machine HTTP adapter for add, role-change and
   deactivate operations on OrganisationMembership and WorkspaceMembership.
   Keep current browser routes and their session/CSRF controls intact.
3. Verify the product signature and acting user's Platform proof separately;
   re-check live Platform authorization on every mutation. Reuse canonical
   Platform store writers rather than adding a second membership rule path.
4. Bind each accepted command to a stable idempotency key and expected
   membership version. Commit the membership mutation, private audit record and
   existing allow-listed outbox event atomically. Keep reasons out of the
   consumer event payload.
5. Add contract and adversarial compatibility fixtures for both `tockrctrl`
   and `tockrims`. Publish the contract and exact endpoint version before
   either consumer replaces its local writer.

## Acceptance criteria

- **B12-AC01:** The command contract explicitly limits writes to the two
  membership facts and the add/role-change/deactivate lifecycle. Organisation
  and Workspace lifecycle, entitlement, assignment and product-role writes
  are rejected as out of scope.
- **B12-AC02:** Every request proves a configured consumer identity and an
  acting Platform user independently. Platform revalidates current target
  scope and role; browser cookies, caller-supplied actor IDs and product roles
  cannot substitute.
- **B12-AC03:** Platform role vocabularies and existing authorization rules
  remain authoritative. Cross-Organisation, cross-Workspace, inactive actor,
  stale assertion and insufficient-role mutations fail closed without state
  change.
- **B12-AC04:** A retry with the same idempotency key and payload returns the
  original result; key reuse with a different payload and a stale expected
  version return explicit conflicts without a second mutation.
- **B12-AC05:** Membership, Platform audit and the existing membership outbox
  event commit atomically. The feed exposes only its existing allow-listed
  fields and never includes mutation reason or credentials.
- **B12-AC06:** CTRL and IMS compatibility fixtures agree on request,
  authorization, result, conflict and failure semantics.
- **B12-AC07:** Browser administration remains covered and unchanged in
  authority. The Platform command endpoints do not activate either product's
  local writer replacement or production cutover.
- **B12-AC08:** Exact-candidate independent review, independent acceptance and
  repository-native validation pass before publication.

## Validation intent

Use repository-supported `format`, `unit`, `integration`, `security`,
`architecture`, `migration` and `race` profiles as selected by the reviewer
and tester. Add focused Platform HTTP tests and one real compatibility fixture
per consumer. Every behavior result must bind to the exact candidate and pass
repository preflight; context failures remain `BLOCKED / NOT RUN`.

```text
candidate=<exact PF-B12-S01 candidate>
authority=B12-AC01..AC08
surface=AUTH|DATA|HIST|CONC|API|OPS|DEP
profile=format,unit,integration,security,architecture,migration,race
command_source=scripts/validation_registry.py and scripts/validate.py
preflight=PASS before behavioral interpretation
```

## Dependencies and stop/go

- PF-B11-S04 is terminal and published.
- The user has authorised the bounded shared-writer decision recorded above.
- Matching CTRL and IMS PD-D5-S01 slices are terminal on the same published
  Platform read-authority contract.
- Any role, lifecycle, actor-proof or last-owner behavior mismatch must be
  recorded and resolved from existing authority; do not guess a new
  cross-product rule.

This slice publishes Platform command capability only. CTRL and IMS D5-S02
remain sequenced after this slice and PD-D6-S01. Production write activation
also remains behind each product's existing reconciliation and cutover
stages, PD-D7-S01 and PD-D7-S02. Ordinary product requests continue to use
local projections and do not require a synchronous Platform lookup.

## Rollback

The new command route is independently disableable. Disabling it must not
change browser administration, canonical membership state, committed audit
history or outbox events. Consumer adapters remain disabled until their
separate D5-S02 and D7 rollout gates pass.
