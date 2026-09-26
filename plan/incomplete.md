# Incomplete Plan Tracker

This is the execution index for every Tockr Platform Foundation (PF) plan and
Platform-owned PD prerequisite that is not terminally implemented. It is derived from the authoritative
[PF programme index](pf-platform-foundation.md) and its linked Slice plans.
Detailed scope, authority and acceptance remain authoritative in the linked
plan; this file controls sequence and provides the shortest safe prompt for
resuming work.

Current inventory: **0 open Platform-owned PD prerequisites** and **0 open PF execution Batches**. Cross-repository PD-D8-S02 is the next ready consumer/authority certification slice.
PF-B1 through PF-B12 are terminal at their accepted Platform scopes. CTRL and
IMS retain separate downstream PD plans.

## How to use this tracker

1. Start with the first unblocked Slice in the queue.
2. Read its linked plan and existing evidence before changing anything.
3. Copy the batch prompt or the first-Slice prompt into a fresh delivery run.
4. When the Slice closes, move its plan to [`completed/`](completed/), remove
   or update its entry here, and promote the next unblocked Slice.

## Completion contract

Every prompt below inherits this contract:

- implement only work that current evidence shows is missing;
- preserve the Platform/CTRL/IMS authority boundary and do not infer identity,
  migration or cutover mappings;
- use behaviour-first tests for code changes and repository-supported
  validation for documentation or tooling changes;
- obtain fresh independent review and independent tester acceptance;
- fix findings sequentially and repeat both independent gates after material
  repairs;
- bind all evidence to the exact candidate being accepted and preserve FAIL or
  BLOCKED / NOT RUN truthfully;
- leave no blocker, bug, unresolved acceptance finding or required evidence row
  open at completion; repair and rerun affected evidence on the exact candidate;
- mark a Slice **Implemented**, move its plan to `completed/`, update
  `docs/implementation/IMPLEMENTED.md`, this tracker, the PF index and
  reconciliation evidence, and publish `main` only when the required gates
  pass.

## Fastest dependency-aware queue

The Platform-owned prerequisite for consumer PD-D7-S01 is sequenced before
consumer cutover. It is not a PF-B13 expansion of terminal foundation scope:

| Queue | Slice | State | Dependency | Plan |
| --- | --- | --- | --- | --- |
| PD-D7-S01-PF | Durable Platform production reconciliation/import boundary | **Terminal** | WP-PD7PF-01..06 accepted on the disposable restored database; CTRL/IMS snapshot/feed and mapping gates passed | [completed plan](completed/pd-d7-s01-platform-production-import.md) |
| PD-D8-S02 | Platform authority integration certification | **Ready / next** | Terminal matched consumer PD-D8-S01 and published Platform read/write contracts | [plan](pd-d8-s02-platform-authority-certification.md) |

Consumer authentication and membership-writer cutover are implemented and terminal in both consumers under PD-D7-S02. PD-D8-S01 product-access/isolation certification is terminal in both consumers. PD-D8-S02 is the next ready cross-repository authority certification; real production activation remains operator-controlled and proceeds only through its accepted cutover and rollback protocol.

PF is the owner-authorised Platform Foundation programme. It establishes
shared identity, tenancy and product-access authority for future CTRL and IMS
consumers. It does not implement CTRL/IMS operational screens, billing,
production-data import or authority cutover.

PF-B11-S04 is now terminally closed below. PF-B11-S01 and its superseding
PF-B11-S01-R1 correction are terminally closed after their exact
contracts, independent gates and local validation passed:

| Queue | Slice | State | Dependency | Plan |
| --- | --- | --- | --- | --- |
| PF-B11 | PF-B11-S04 — Security, operability and consumer-readiness certification | **Terminal** | S03 terminal; authenticated v2 feed and resynchronisation API | [completed plan](completed/pf-b11-s04-certification.md) |
| PF-B12 | PF-B12-S01 — Versioned shared membership command API | **Terminal** | Certification candidate `87872417600a750a6cad0a81d2106c0f56ce78e6`; review, acceptance and local validation passed | [completed plan](completed/pf-b12-s01-membership-command-api.md) |

PF-B1 through PF-B11 form the complete terminal inventory:

| Batch | Slices | State | Depends on |
| --- | --- | --- | --- |
| PF-B1 | S01, S02, S03 | **Terminal** | S01 → S02 → S03 |
| PF-B2 | S01, S02 | **Terminal** | PF-B1-S03; S01 → S02 |
| PF-B3 | S01, S02 | **Terminal** | PF-B1-S03; S01 → S02 |
| PF-B4 | S01, S02 | **Terminal** | PF-B1-S03; S01 → S02 |
| PF-B5 | S01, S02 | **Terminal** | PF-B2-S02 + PF-B3-S02 + PF-B4-S02; S01 → S02 |
| PF-B6 | S01, S02 | **Terminal** | PF-B5-S02; S01 → S02 |
| PF-B7 | S01, S02 | **Terminal** | PF-B5-S02; S01 → S02 |
| PF-B8 | S01, S02 | **Terminal** | PF-B6-S02 + PF-B7-S02; S01 → S02 |
| PF-B9 | S01, S02 | **Terminal** | PF-B6-S02 + PF-B7-S02; S01 → S02 |
| PF-B10 | S01, S01-R1, S02 | **Terminal** | PF-B8-S02 + PF-B9-S02; S01 → S01-R1 → S02 |
| PF-B11 | S01, S01-R1, S02, S03 and S04 | **Terminal** | S01 → S01-R1 → S02 → S03 → S04 |

The three authority branches after PF-B1-S03 are sequential within each
branch; they are not parallel implementation permission. Lane 1 prepares every
Batch, Lane 2 implements one Slice at a time, and Lane 3 certifies the Batch.

## Supporting programme index

The full PF dependency spine, authority boundary and stop conditions are in
[`pf-platform-foundation.md`](pf-platform-foundation.md). Slice-level acceptance and
routing remain in the linked active or completed plan files.

## Copy-ready next Batch prompt

This prompt follows the canonical [Standard PF Batch execution
prompt](pf-platform-foundation.md#standard-pf-batch-execution-prompt) in the PF
programme plan.

> Deliver the next active **Tockr Platform Foundation Batch** from
> [`plan/incomplete.md`](incomplete.md). Identify the first Batch whose
> dependencies are terminal and whose queue contains a Ready Slice. Read
> [`plan/pf-platform-foundation.md`](pf-platform-foundation.md) and every linked Slice
> plan belonging to that Batch before changing anything. Execute the Batch's
> Slices strictly in dependency order: complete one Slice, pass its
> independent review, independent tester acceptance and exact-candidate local
> validation, close it out, and only then promote the next Slice. Do not
> implement Slices in parallel, skip a gate, or begin the next Slice before the
> previous Slice is terminally closed. Implement only each Slice's authorised
> scope and preserve the Platform/CTRL/IMS boundary. Stop and record a
> truthful BLOCKED / NOT RUN or unresolved-authority state if a dependency,
> source conflict or acceptance condition cannot be proved. After every Slice
> in the Batch passes, perform Batch certification, update
> `plan/incomplete.md`, the PF index, `docs/implementation/IMPLEMENTED.md`,
> reconciliation evidence and `plan/completed/`, then publish `main` only if
> the required gates pass. Do not implement work from a later Batch or any
> scope outside the selected Batch.

## Copy-ready first Slice prompt

> No open Platform Slice remains in PF-B12. The Platform command contract is
> terminal. Deliver the Platform-owned PD-D7-S01-PF prerequisite from
> [`plan/completed/pd-d7-s01-platform-production-import.md`](completed/pd-d7-s01-platform-production-import.md)
> before real CTRL/IMS import or consumer cutover. Then continue with the
> separately governed CTRL/IMS consumer plans.

## Closeout update

PF-B7-S01 and PF-B7-S02 are terminal and PF-B7 is certified. Batch
reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b7-batch-certification.md`](../docs/implementation/audits/pf-b7-batch-certification.md).
S02 reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b7-s02-projections.md`](../docs/implementation/audits/pf-b7-s02-projections.md).
PF-B8-S01 and PF-B8-S02 are terminal and PF-B8 is certified. Batch
reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b8-batch-certification.md`](../docs/implementation/audits/pf-b8-batch-certification.md).
S01 reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b8-s01-platform-shell.md`](../docs/implementation/audits/pf-b8-s01-platform-shell.md),
and S02 reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b8-s02-administration.md`](../docs/implementation/audits/pf-b8-s02-administration.md).
PF-B9-S01 and PF-B9-S02 are terminally reconciled and PF-B9 is certified.
PF-B10-S01 is terminal and its published candidate's browser-discovered
runtime asset packaging regression was repaired by PF-B10-S01-R1. R1's
reconciliation, independent review and independent acceptance are recorded in
[`docs/implementation/audits/pf-b10-s01-r1-runtime-assets.md`](../docs/implementation/audits/pf-b10-s01-r1-runtime-assets.md),
with the linked audit files. PF-B10-S02 final certification and PF-B10 Batch
certification are now terminal.
The original reconciliation is recorded in
[`docs/implementation/audits/pf-b10-s01-runtime-hardening.md`](../docs/implementation/audits/pf-b10-s01-runtime-hardening.md).
PF-B10 is terminally certified; final evidence is recorded in
[`docs/implementation/audits/pf-b10-batch-certification.md`](../docs/implementation/audits/pf-b10-batch-certification.md).
PF-B11-S01 is now terminally implemented and reconciled in
[`plan/completed/pf-b11-s01-read-authority-contract.md`](completed/pf-b11-s01-read-authority-contract.md),
with independent review and acceptance in the linked audit files. PF-B11-S01-R1
is now terminally implemented and reconciled in
[`plan/completed/pf-b11-s01-r1-seed-provenance-v2.md`](completed/pf-b11-s01-r1-seed-provenance-v2.md),
with independent review and acceptance recorded in the linked audit files.
PF-B11-S02 is now terminally implemented and reconciled in
[`plan/completed/pf-b11-s02-durable-snapshot.md`](completed/pf-b11-s02-durable-snapshot.md),
with independent review and acceptance recorded in the linked audit files.
PF-B11-S03 is now terminally implemented and reconciled in
[`plan/completed/pf-b11-s03-feed-and-api.md`](completed/pf-b11-s03-feed-and-api.md),
with independent review and acceptance recorded in the linked audit files.
PF-B11-S04 is now terminally certified and reconciled in
[`plan/completed/pf-b11-s04-certification.md`](completed/pf-b11-s04-certification.md),
with Batch certification in
[`docs/implementation/audits/pf-b11-batch-certification.md`](../docs/implementation/audits/pf-b11-batch-certification.md).
PF-B11 is terminal; no later Batch was started.
On 2026-09-21 the user authorised the bounded shared-membership writer scope
for CTRL and IMS. PF-B12-S01 is now terminally implemented and reconciled in
[`completed/pf-b12-s01-membership-command-api.md`](completed/pf-b12-s01-membership-command-api.md),
with independent review, tester acceptance and Batch certification in the
linked audit files. Consumer D5-S02 remains disabled/local-mode work in the
consumer repositories.
PF-B6 has now been certified as a terminal Batch; its certification is
recorded in [`docs/implementation/audits/pf-b6-batch-certification.md`](../docs/implementation/audits/pf-b6-batch-certification.md).
For every terminal Slice:

1. update `docs/implementation/IMPLEMENTED.md`;
2. move the immutable plan into `completed/`;
3. remove or update the Slice entry and promote the next unblocked item here;
4. update `plan/README.md`, the PF index, reconciliation evidence and any
   completed-plan totals;
5. verify the exact committed/published candidate and clean `main` when
   publication is required.

Do not close a Slice because its plan was written, because a reviewer made a
proposal, or because a validation route was unavailable. Preserve those states
as planning or BLOCKED / NOT RUN evidence.
