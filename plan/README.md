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
authority Batch, the complete PF-B4 Workspace authority Batch and the complete
PF-B5 product-access Batch and PF-B6 product assertion/consumer Batch; the next
active queue item is PF-B7-S01.
PF-B6 Batch certification is recorded in
[`docs/implementation/audits/pf-b6-batch-certification.md`](../docs/implementation/audits/pf-b6-batch-certification.md).

Plans use the repository delivery model:

```text
Priority → Batch → Slice
Lane 1 preparation → Lane 2 sequential implementation → Lane 3 certification
```
