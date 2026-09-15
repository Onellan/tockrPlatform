# Tockr Platform plans

`active/` contains the current PF Slice plans. `completed/` will contain
immutable terminal plans after the delivery gates pass. Do not edit a terminal
plan to absorb later work; create or supersede an active plan instead.

Plans use the repository delivery model:

```text
Priority → Batch → Slice
Lane 1 preparation → Lane 2 sequential implementation → Lane 3 certification
```
