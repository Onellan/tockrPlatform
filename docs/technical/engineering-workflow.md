# Platform engineering workflow

## Three lanes

```text
LANE 1 — Batch preparation
    authority, dependencies, design seams, validation context and rollback

LANE 2 — Sequential Slice implementation
    one authorised Slice at a time, behaviour-first evidence and narrow review

LANE 3 — Batch validation/certification
    independent batch architecture review, tester acceptance and local profile
```

Slices do not run in parallel when they share authority or migration order.

## Gates

```text
plan → implement → independent engineering review → independent acceptance
     → exact candidate local/release validation → terminal record
```

An implementation candidate is not an acceptance verdict. A reviewer does not
fix its own findings; a tester does not inherit implementer evidence. Repairs
re-enter the affected gates. Full repository certification is reserved for
Batch and final boundaries rather than repeated for every small Slice.

## Planning authority

The programme index establishes Priority/Batch/Slice order. Each Slice plan is
the implementation boundary and contains objective, authority, evidence,
affected seams, ordered work, migration/security impact, acceptance, tests,
dependencies, stop/go and rollback. Plans do not implement application code.

## Publication boundary

This foundation task is explicitly authorised to push planning/setup work to
Platform `main`. Future runtime delivery still requires the applicable delivery
authority and exact-candidate evidence. CTRL/IMS migration or authority cutover
never follows from a Platform commit alone.
