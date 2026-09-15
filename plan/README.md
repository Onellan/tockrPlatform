# Tockr Platform plans

[`incomplete.md`](incomplete.md) is the execution index for every PF plan that
is not terminally implemented. [`active/`](active/) contains the current PF
Slice plans. [`completed/`](completed/) contains immutable terminal plans after
the delivery gates pass. Do not edit a terminal plan to absorb later work;
create or supersede an active plan instead.

The active programme index is
[`active/priority-pf.md`](active/priority-pf.md). The Platform tracker keeps
the same queue, closeout and evidence conventions as TockrIMS and TockrCTRL
while retaining Platform's `Priority → Batch → Slice` structure.

Plans use the repository delivery model:

```text
Priority → Batch → Slice
Lane 1 preparation → Lane 2 sequential implementation → Lane 3 certification
```
