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
Batch, PF-B7-S01/S02, PF-B8-S01/S02, PF-B9-S01/S02 and PF-B10-S01; PF-B7,
PF-B8 and PF-B9 are certified; PF-B10-S01-R1 is terminal and PF-B10-S02 is
PF-B10-S02 is terminal and the original PF Foundation queue has no remaining
open item. PF-B11 is the separately authorised forward extension described
below; its S01, S01-R1 and S02 slices are terminal and S03 is now Ready.
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
new explicitly authorised forward Batch that supplies the missing versioned,
bounded and fail-closed Platform read-authority contract required before CTRL
or IMS may start PD-D5-S01. PF-B11-S01, PF-B11-S01-R1 and PF-B11-S02 are
terminal; S02 delivered the durable v2 snapshot/source-cursor seam and S03 is
now **Ready**. S04 remains unrun. The
Batch and remaining Slice plans are:

- [`pf-b11-shared-read-authority.md`](pf-b11-shared-read-authority.md)
- [`completed/pf-b11-s01-read-authority-contract.md`](completed/pf-b11-s01-read-authority-contract.md)
- [`completed/pf-b11-s01-r1-seed-provenance-v2.md`](completed/pf-b11-s01-r1-seed-provenance-v2.md)
- [`completed/pf-b11-s02-durable-snapshot.md`](completed/pf-b11-s02-durable-snapshot.md)
- [`active/pf-b11-s03-feed-and-api.md`](active/pf-b11-s03-feed-and-api.md)
- [`active/pf-b11-s04-certification.md`](active/pf-b11-s04-certification.md)

PF-B11 does not reopen terminal plans, implement CTRL/IMS cutover or move
product roles, billing or product data into Platform.
