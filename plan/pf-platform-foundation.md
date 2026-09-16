# Priority PF — Tockr Platform Foundation

**Status:** Planned / active owner-authorised programme; PF-B1 through PF-B8 are terminal, and PF-B9-S01 is Ready.
**Authority:** [Platform architecture](../architecture.md), [Platform
ownership and boundaries](../docs/architecture/platform-ownership-and-boundaries.md),
[Platform contract v1](../docs/contracts/platform-contract-v1.md), and the
[Platform engineering workflow](../docs/technical/engineering-workflow.md).

PF establishes the shared Platform authority that future TockrCTRL and
TockrIMS consumers will use for identity, tenancy and product access. This
file is the programme index and execution contract; each linked Slice plan is
the authoritative implementation boundary and acceptance detail.

## Objective

Deliver a repository-native Platform foundation that freezes ownership,
contracts, runtime seams and certification order before application behaviour
or CTRL/IMS authority migration is implemented.

The programme must preserve local product ownership, fail closed at shared
security boundaries, keep historical facts truthful and make every later
identity, entitlement, assertion, projection and migration decision
independently provable.

## Authority map

| Decision surface | Authority | PF boundary |
| --- | --- | --- |
| Platform ownership and prohibited product responsibilities | [Platform ownership and boundaries](../docs/architecture/platform-ownership-and-boundaries.md) | Platform owns shared identity, tenancy, entitlement and assignment; CTRL/IMS retain product domains and roles |
| Cross-repository records, access predicate and assertion envelope | [Platform contract v1](../docs/contracts/platform-contract-v1.md) | Contract versioning, compatibility and fail-closed validation are mandatory |
| Target runtime, package responsibilities and non-negotiable boundaries | [Technical architecture](../architecture.md) | Go modular monolith, local SQLite, narrow capability seams and no shared product database |
| Events, inboxes, projections and reconciliation | [Events and projection contract](../docs/architecture/events-and-projection-contract.md) | Delivery is idempotent and ordered; projections do not transfer authority silently |
| Security, health, readiness and hardened runtime | [Security runtime contract](../docs/architecture/security-runtime-contract.md) | Secrets and tenant data stay out of health output; runtime controls are server-side |
| Source conflict and inherited semantics | [Source alignment matrix](../docs/architecture/source-alignment-matrix.md) | CTRL/IMS evidence is tied to exact inspected SHAs; unresolved conflicts stop delivery |
| Delivery lanes, gates and publication | [Engineering workflow](../docs/technical/engineering-workflow.md) | Plan, implementation, independent review, independent acceptance and exact-candidate validation remain separate gates |

## Current state and programme baseline

| Source | Exact SHA |
| --- | --- |
| TockrCTRL current `main` inspected | `47d20f29ad36e59d44b1154ebcde1ed5a99905f6` |
| TockrIMS current `main` inspected | `aa5de6c4672114c35b3f31fdc3177189a78c9b5d` |
| Platform starting `main` | `55301d8103feee7c0c919703ae01457af0980dcd` |

The baseline records the source evidence used to create PF; it is not an
acceptance candidate. At the current foundation state Platform contains the
terminal repository-control-plane foundation plus the PF-B2-S01 Platform-owned
identity and authentication runtime, SQLite migration ledger, secure sessions,
MFA/recovery controls, Organisation lifecycle, canonical membership authority
and narrow administration seams with focused acceptance evidence. PF-B4-S01
adds Organisation-owned Workspace lifecycle, generic membership history and
entry seams and PF-B4-S02 adds the reusable active Workspace scope guard.
PF-B5, PF-B6 and PF-B7 are terminally implemented. PF-B8-S01 delivers the
shared server-rendered Platform shell, Organisation/Workspace selectors and
access-gated product launcher; PF-B8-S02 delivers the initial Organisation,
Workspace and System Admin surfaces. PF-B9-S01 is the next dependency-ready
Slice.
Containers, production-data import
and CTRL/IMS authority cutover remain out of scope.
PF-B6 Batch certification is recorded in
[`docs/implementation/audits/pf-b6-batch-certification.md`](../docs/implementation/audits/pf-b6-batch-certification.md).
PF-B7-S01 reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b7-s01-events.md`](../docs/implementation/audits/pf-b7-s01-events.md).
PF-B7-S02 reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b7-s02-projections.md`](../docs/implementation/audits/pf-b7-s02-projections.md).
PF-B7 Batch certification is recorded in
[`docs/implementation/audits/pf-b7-batch-certification.md`](../docs/implementation/audits/pf-b7-batch-certification.md).
PF-B8-S01 reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b8-s01-platform-shell.md`](../docs/implementation/audits/pf-b8-s01-platform-shell.md),
with independent review and acceptance in the linked audit files.
PF-B8-S02 reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b8-s02-administration.md`](../docs/implementation/audits/pf-b8-s02-administration.md).
PF-B8 Batch certification is recorded in
[`docs/implementation/audits/pf-b8-batch-certification.md`](../docs/implementation/audits/pf-b8-batch-certification.md).

## Plan contract

- The Priority → Batch → Slice order in this file controls programme sequence.
- Each linked Slice plan controls its own objective, authority, current
  evidence, affected seams, ordered work, migration/security impact,
  acceptance, tests, dependencies, stop/go conditions and rollback.
- No Slice may silently absorb a later Slice, reopen a terminal plan or infer
  an identity/migration mapping from incomplete evidence.
- A plan, review conclusion or unavailable validation route is not
  implementation or acceptance evidence.
- Material repairs re-enter independent review and independent acceptance;
  evidence is always bound to the exact candidate under review.
- The tracker and implementation ledger are reconciled in the same closeout
  change as a terminal Slice or Batch.

## Programme-wide non-goals

PF does not implement CTRL/IMS operational screens, product-domain Projects,
product-specific roles, billing/payment authority, production-data import,
automatic identity mapping, shared databases, service decomposition or
authority cutover. Any such work requires explicit forward authority in the
relevant consumer repository and a compatible Platform contract.

## Three-lane execution model

```text
LANE 1 — Batch preparation
    authority, dependencies, design seams, validation context and rollback

LANE 2 — Sequential Slice implementation
    one authorised Slice at a time, behaviour-first evidence and narrow review

LANE 3 — Batch validation and certification
    independent architecture review, independent tester acceptance and local profile
```

Slices do not run in parallel when they share authority, migration order or
contract decisions. The PF-B1 authority foundation is therefore S01 → S02 →
S03, not three parallel work items.

## Gates and validation contract

```text
plan → implement → independent engineering review → independent acceptance
     → exact-candidate local/release validation → terminal record
```

Use the repository validation authority through `scripts/validate.py` and its
registry. The current foundation candidate has a Go/SQLite runtime but no
Dockerfile, so runtime profiles are executable while container build profiles
remain `NOT_APPLICABLE`; unavailable profiles must never be represented as
runtime PASS evidence.
Context failures such as `INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`,
`FIXTURE_FAIL` and `PREREQUISITE_FAIL` are reported truthfully and do not
justify application repair.

GitHub-hosted CI is optional publication evidence unless a Slice plan names it
as an acceptance requirement. Local validation, independent review and
independent tester acceptance remain separate gates.

## Ordered queue

| Batch | Purpose | Slices | Status |
| --- | --- | --- | --- |
| PF-B1 | Repository, standards and architecture foundation | S01–S03 | **Terminal** |
| PF-B2 | Identity and authentication | S01–S02 | **Terminal** |
| PF-B3 | Organisation authority | S01–S02 | **Terminal** |
| PF-B4 | Workspace authority | S01–S02 | **Terminal** |
| PF-B5 | Product catalogue and product access | S01–S02 | **Terminal** |
| PF-B6 | Product assertion and consumer contract | S01–S02 | **Terminal** |
| PF-B7 | Events and local projection support | S01–S02 | **Terminal** |
| PF-B8 | Platform administration UI | S01–S02 | **Terminal** |
| PF-B9 | CTRL/IMS reconciliation and migration tooling | S01–S02 | S01 **Ready**; S02 Planned |
| PF-B10 | Security, runtime and final certification | S01–S02 | Planned |

Total: **10 Batches, 21 Slices**. PF-B1 has three foundation Slices because
the current CTRL/IMS SQLite pool policy conflict must be resolved before
runtime implementation; collapsing it would hide an implementation authority
decision.

## Dependency spine

```text
PF-B1-S01 → PF-B1-S02 → PF-B1-S03
                         ├→ PF-B2-S01 → PF-B2-S02
                         ├→ PF-B3-S01 → PF-B3-S02
                         └→ PF-B4-S01 → PF-B4-S02
PF-B2-S02 + PF-B3-S02 + PF-B4-S02 → PF-B5-S01 → PF-B5-S02
PF-B5-S02 → PF-B6-S01 → PF-B6-S02
PF-B5-S02 → PF-B7-S01 → PF-B7-S02
PF-B6-S02 + PF-B7-S02 → PF-B8-S01 → PF-B8-S02
PF-B6-S02 + PF-B7-S02 → PF-B9-S01 → PF-B9-S02
PF-B8-S02 + PF-B9-S02 → PF-B10-S01 → PF-B10-S02
```

The three authority branches after B1-S03 are sequential within each branch;
they are not parallel implementation permission. Lane 1 must prepare every
Batch, Lane 2 implements one Slice at a time, and Lane 3 certifies the Batch.

## Slice plans

| Slice | Plan |
| --- | --- |
| PF-B1-S01 | [repository, standards, agents and validation — terminal](completed/pf-b1-s01-repository-foundation.md) |
| PF-B1-S02 | [ownership and shared contracts — terminal](completed/pf-b1-s02-ownership-contracts.md) |
| PF-B1-S03 | [runtime, persistence and presentation foundation — terminal](completed/pf-b1-s03-runtime-foundation.md) |
| PF-B2-S01 | [User and authentication — terminal](completed/pf-b2-s01-user-authentication.md) |
| PF-B2-S02 | [sessions, MFA, recovery and revocation — terminal](completed/pf-b2-s02-sessions-security.md) |
| PF-B3-S01 | [Organisation authority — terminal](completed/pf-b3-s01-organisation-authority.md) |
| PF-B3-S02 | [Organisation administration seams — terminal](completed/pf-b3-s02-organisation-administration.md) |
| PF-B4-S01 | [Workspace authority — terminal](completed/pf-b4-s01-workspace-authority.md) |
| PF-B4-S02 | [Workspace access and scope guard — terminal](completed/pf-b4-s02-workspace-access.md) |
| PF-B5-S01 | [Product catalogue and entitlements — terminal](completed/pf-b5-s01-product-catalogue.md) |
| PF-B5-S02 | [User assignment and effective access — terminal](completed/pf-b5-s02-product-access.md) |
| PF-B6-S01 | [Assertion signing and verification contract — terminal](completed/pf-b6-s01-assertions.md) |
| PF-B6-S02 | [Consumer handoff and compatibility — terminal](completed/pf-b6-s02-consumer-contract.md) |
| PF-B7-S01 | [Platform events and outbox — terminal](completed/pf-b7-s01-events.md) |
| PF-B7-S02 | [Projection inbox and reconciliation — terminal](completed/pf-b7-s02-projections.md) |
| PF-B8-S01 | [Layouts, selectors and launcher — terminal](completed/pf-b8-s01-platform-shell.md) |
| PF-B8-S02 | [Administration UI — terminal](completed/pf-b8-s02-administration-ui.md) |
| PF-B9-S01 | [Reconciliation inventory and mapping](active/pf-b9-s01-reconciliation.md) |
| PF-B9-S02 | [Dry-run/import and rollback tooling](active/pf-b9-s02-migration-tooling.md) |
| PF-B10-S01 | [Security and hardened runtime](active/pf-b10-s01-runtime-hardening.md) |
| PF-B10-S02 | [Final local certification](active/pf-b10-s02-final-certification.md) |

## Programme work packages

The programme-level work packages route execution; the linked Slice plans
contain the detailed implementation steps.

1. Prepare each Batch from its authority, dependencies, design seams,
   validation context and rollback conditions.
   Route: kind=other; risk=H[AUTH,DEP,DOC,GOV]
2. Implement the Batch's Slices sequentially, with behaviour-first evidence,
   independent review and independent tester acceptance after each Slice.
   Route: kind=other; risk=H[DATA,AUTH,CONC,OPS,API,UI,DEP]
3. Certify the completed Batch, reconcile the exact candidate and promote only
   the next dependency-ready Batch.
   Route: kind=other; risk=H[OPS,GOV,HIST,DOC,DEPLOY]

## Programme acceptance ledger

Every row below is satisfied only by the linked Slice acceptance criteria plus
the independent Batch certification gate. It is a programme-level map, not a
replacement for the Slice-level acceptance detail.

| Batch | Required programme outcome | Evidence owner |
| --- | --- | --- |
| PF-B1 | Repository controls, ownership contracts and runtime/persistence/presentation foundation are explicit and implementation-ready | PF-B1-S01–S03 plans and Batch certification |
| PF-B2 | User identity, authentication, sessions, MFA, recovery and revocation are Platform-owned and fail closed | PF-B2-S01–S02 plans and Batch certification |
| PF-B3 | Organisation lifecycle, membership and administration authority are explicit and scoped | PF-B3-S01–S02 plans and Batch certification |
| PF-B4 | Workspace ownership, membership, access scope and isolation are explicit | PF-B4-S01–S02 plans and Batch certification |
| PF-B5 | Product catalogue, Organisation entitlement, User assignment and effective access remain separate decisions | PF-B5-S01–S02 plans and Batch certification |
| PF-B6 | Signed assertions and versioned consumer compatibility are defined and fail closed | PF-B6-S01–S02 plans and Batch certification |
| PF-B7 | Versioned events, transactional outbox, inbox identity and projection ordering are explicit | PF-B7-S01–S02 plans and Batch certification |
| PF-B8 | Platform shell and administration UI preserve server-side authority and source-owned presentation rules | PF-B8-S01–S02 plans and Batch certification |
| PF-B9 | CTRL/IMS reconciliation, mapping, dry-run/import and rollback are explicit without guessed identity | PF-B9-S01–S02 plans and Batch certification |
| PF-B10 | Security hardening, runtime controls and final exact-candidate certification are complete | PF-B10-S01–S02 plans and final certification |

## Standard PF Batch execution prompt

```text
Deliver the next active Tockr Platform Foundation Batch from
plan/incomplete.md. Identify the first Batch whose dependencies are terminal
and whose queue contains a Ready Slice. Read this programme plan and every
linked Slice plan belonging to that Batch before changing anything. Execute
the Batch's Slices strictly in dependency order: complete one Slice, obtain
independent engineering review, independent tester acceptance and
exact-candidate local validation, close it out, and only then promote the next
Slice. Do not implement Slices in parallel, skip a gate, or begin the next
Slice before the previous Slice is terminally closed. Implement only the
selected Slice scope and preserve the Platform/CTRL/IMS boundary. Stop and
record truthful BLOCKED / NOT RUN or unresolved-authority evidence if a
dependency, source conflict or acceptance condition cannot be proved. After
all Slices in the Batch pass, perform Batch certification, reconcile the exact
candidate, move terminal plans to plan/completed/, update plan/incomplete.md,
the PF index and docs/implementation/IMPLEMENTED.md, and publish main only
when the required gates pass. Do not implement a later Batch.
```

## Programme stop conditions

Stop PF when a source semantic conflict is unresolved, a product boundary is
ambiguous, an identity mapping would guess, a design seam would scatter
authority, an acceptance condition cannot be proved locally, or a proposed
change would require CTRL/IMS migration or authority cutover outside this
programme. Record the blocker and preserve the queue state; do not convert
BLOCKED / NOT RUN into PASS.

## Completion definition and publication boundary

PF is complete only when PF-B10-S02 has passed its independent review,
independent tester acceptance, exact-candidate local validation and final
certification, with every prior Batch terminally reconciled. The final record
must bind the accepted candidate, plan paths, implementation ledger and
validation evidence, and leave `main` clean when publication is required.

Platform publication does not authorise CTRL/IMS migration, production-data
import or authority cutover. Those actions require their own forward plans,
compatibility evidence, rollback rules and independent acceptance in the
consumer repositories.
