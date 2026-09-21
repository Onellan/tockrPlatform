# Tockr Platform plans

[`incomplete.md`](incomplete.md) is the execution index for every PF plan that
is not terminally implemented. [`active/`](active/) contains the current PF
Slice plans. [`completed/`](completed/) contains immutable terminal plans after
the delivery gates pass. Do not edit a terminal plan to absorb later work;
create or supersede an active plan instead.

The programme index is
[`pf-platform-foundation.md`](pf-platform-foundation.md). The Platform tracker keeps
the same queue, closeout and evidence conventions as TockrIMS and TockrCTRL
while retaining Platform's `Priority → Batch → Slice` structure.

The completed ledger currently contains the terminal PF-B1 foundation Slices,
the PF-B2 identity/authentication Batch, the complete PF-B3 Organisation
authority Batch, the complete PF-B4 Workspace authority Batch, the complete
PF-B5 product-access Batch, the complete PF-B6 product assertion/consumer
Batch, PF-B7-S01/S02, PF-B8-S01/S02, PF-B9-S01/S02 and PF-B10. PF-B7, PF-B8,
PF-B9, PF-B10 and PF-B11 are terminally certified. PF-B12 is the active,
separately authorised forward extension described below.
PF-B7 Batch certification is recorded in
[`docs/implementation/audits/pf-b7-batch-certification.md`](../docs/implementation/audits/pf-b7-batch-certification.md).
PF-B6 Batch certification is recorded in
[`docs/implementation/audits/pf-b6-batch-certification.md`](../docs/implementation/audits/pf-b6-batch-certification.md).
PF-B8-S01 reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b8-s01-platform-shell.md`](../docs/implementation/audits/pf-b8-s01-platform-shell.md).
PF-B8-S02 reconciliation evidence and Batch certification are recorded in
[`docs/implementation/audits/pf-b8-s02-administration.md`](../docs/implementation/audits/pf-b8-s02-administration.md)
and [`docs/implementation/audits/pf-b8-batch-certification.md`](../docs/implementation/audits/pf-b8-batch-certification.md).
PF-B9-S01 reconciliation evidence is recorded in
[`docs/implementation/audits/pf-b9-s01-reconciliation.md`](../docs/implementation/audits/pf-b9-s01-reconciliation.md).
PF-B9-S02 reconciliation evidence and Batch certification are recorded in
[`docs/implementation/audits/pf-b9-s02-migration-tooling.md`](../docs/implementation/audits/pf-b9-s02-migration-tooling.md)
and [`docs/implementation/audits/pf-b9-batch-certification.md`](../docs/implementation/audits/pf-b9-batch-certification.md).

Plans use the repository delivery model:

```text
Priority → Batch → Slice
Lane 1 preparation → Lane 2 sequential implementation → Lane 3 certification
```

## Forward PF-B11 — Shared read authority and consumer projection source

PF-B1 through PF-B10 remain terminal historical Foundation scope. PF-B11 is a
new explicitly authorised and terminally certified Batch that supplies the
versioned, bounded and fail-closed Platform read-authority contract required
before CTRL or IMS may start PD-D5-S01. PF-B11-S01, S01-R1, S02, S03 and S04
are terminal. The Batch and completed Slice plans are:

- [`pf-b11-shared-read-authority.md`](pf-b11-shared-read-authority.md)
- [`completed/pf-b11-s01-read-authority-contract.md`](completed/pf-b11-s01-read-authority-contract.md)
- [`completed/pf-b11-s01-r1-seed-provenance-v2.md`](completed/pf-b11-s01-r1-seed-provenance-v2.md)
- [`completed/pf-b11-s02-durable-snapshot.md`](completed/pf-b11-s02-durable-snapshot.md)
- [`completed/pf-b11-s03-feed-and-api.md`](completed/pf-b11-s03-feed-and-api.md)
- [`completed/pf-b11-s04-certification.md`](completed/pf-b11-s04-certification.md)

PF-B11 does not reopen terminal plans, implement CTRL/IMS cutover or move
product roles, billing or product data into Platform.

## Forward PF-B12 — Shared membership command authority

PF-B12 is a separately authorised forward extension for the user-approved
OrganisationMembership and generic WorkspaceMembership writer scope. Its
Platform-owned command contract and API must be delivered before CTRL/IMS
replace their local membership writers. PF-B12 does not authorize production
consumer cutover, migration/import, product-role changes, entitlement or
assignment writes.

- [PF-B12 Batch plan](pf-b12-shared-membership-commands.md)
- [PF-B12-S01 — Versioned shared membership command API](active/pf-b12-s01-membership-command-api.md)
