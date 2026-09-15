# Incomplete Plan Tracker

This is the execution index for every Tockr Platform Foundation (PF) plan that
is not terminally implemented. It is derived from the authoritative
[PF programme index](active/priority-pf.md) and its linked Slice plans.
Detailed scope, authority and acceptance remain authoritative in the linked
plan; this file controls sequence and provides the shortest safe prompt for
resuming work.

Current inventory: **21 open execution plans** across **10 Batches**. The
Platform repository contains foundation planning and delivery scaffolding only
at this stage; this tracker must not be read as runtime implementation
evidence.

## How to use this tracker

1. Start with the first unblocked Slice in the queue.
2. Read its linked plan and existing evidence before changing anything.
3. Copy the Slice prompt into a fresh delivery run.
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
- mark a Slice **Implemented**, move its plan to `completed/`, update
  `docs/implementation/IMPLEMENTED.md`, this tracker, the PF index and
  reconciliation evidence, and publish `main` only when the required gates
  pass.

## Fastest dependency-aware queue

PF is the owner-authorised Platform Foundation programme. It establishes
shared identity, tenancy and product-access authority for future CTRL and IMS
consumers. It does not implement CTRL/IMS operational screens, billing,
production-data import or authority cutover.

The first executable item is:

| Queue | Slice | State | Dependency | Plan |
| --- | --- | --- | --- | --- |
| 1 | PF-B1-S01 | **Ready** | None | [repository, standards, agents and validation foundation](active/pf-b1-s01-repository-foundation.md) |

All later Slices remain dependency-bound. The complete inventory is:

| Batch | Slices | State | Depends on |
| --- | --- | --- | --- |
| PF-B1 | S01, S02, S03 | S01 Ready; S02/S03 Planned | S02 follows S01; S03 follows S02 |
| PF-B2 | S01, S02 | Planned | PF-B1-S03 |
| PF-B3 | S01, S02 | Planned | PF-B1-S03 |
| PF-B4 | S01, S02 | Planned | PF-B1-S03 |
| PF-B5 | S01, S02 | Planned | PF-B2-S02 + PF-B3-S02 + PF-B4-S02 |
| PF-B6 | S01, S02 | Planned | PF-B5-S02 |
| PF-B7 | S01, S02 | Planned | PF-B5-S02 |
| PF-B8 | S01, S02 | Planned | PF-B6-S02 + PF-B7-S02 |
| PF-B9 | S01, S02 | Planned | PF-B6-S02 + PF-B7-S02 |
| PF-B10 | S01, S02 | Planned | PF-B8-S02 + PF-B9-S02 |

The three authority branches after PF-B1-S03 are sequential within each
branch; they are not parallel implementation permission. Lane 1 prepares every
Batch, Lane 2 implements one Slice at a time, and Lane 3 certifies the Batch.

## Supporting programme index

The full PF dependency spine, authority boundary and stop conditions are in
[`active/priority-pf.md`](active/priority-pf.md). Slice-level acceptance and
routing remain in the linked active plan files.

## Copy-ready next prompt

> Deliver **PF-B1-S01 — Repository, standards, agents and validation
> foundation** from
> [`plan/active/pf-b1-s01-repository-foundation.md`](active/pf-b1-s01-repository-foundation.md).
> Re-read the plan, verify the current candidate and repository state, implement
> only its authorised scope, then obtain independent review and independent
> acceptance. Repair findings sequentially, rerun the required local validation
> against the exact final candidate, and update the PF tracker/index and
> implementation reconciliation only if every required gate passes. Do not
> implement PF-B1-S02 or any runtime functionality in this Slice.

## Closeout update

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
