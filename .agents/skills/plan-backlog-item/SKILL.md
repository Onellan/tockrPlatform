---
name: plan-backlog-item
description: Plan one active tockrPlatform backlog item into a repository-grounded execution plan without implementing it.
---

# Plan a tockrPlatform Backlog Item

Plan exactly one active item. Before planning, read `docs/implementation/IMPLEMENTED.md` and verify the target is not terminal.

An Implemented slice and its historical plan remain terminal. Do not reopen, edit or reinterpret that plan as active work. If the current active item exposes a defect, regression, unsafe assumption, stale documentation or maintainability problem in code delivered by an Implemented slice, record the problem as current-state evidence in the **current active plan** and own the forward fix there. Preserve the old slice's historical acceptance behaviour with regression evidence. If the intended change materially changes the delivered product contract rather than repairing it, require new or explicitly superseding product/backlog authority.

## Validation planning

Use `test-execution` whenever the plan defines implementation, regression, review or acceptance evidence.

Every ordered work package must contain exactly one compact routing signature:

```text
Route: kind=<routine|defect|migration|authorization|ui|other>; risk=<R|E|H>[CODE,...]
```

Use `R` only for clearly bounded work. E/H and Risk R packages with DATA,
AUTH, FIN, HIST, CONC, OPS, API, PERF, DEPLOY, DEP, GOV or DOC must not be
presented as medium-eligible. The signature is routing metadata, not product
authority; the implementer re-checks it independently.

The planner decides **what** each AC/risk surface needs proving. The repository decides **how** supported proof is executed through `scripts/validate.py` / `scripts/validation_registry.py`.

- prefer a named registry profile or focused repository target instead of hand-writing executable commands;
- if the repository has no supported route, label a proposed command `INFERRED` and require its target/prerequisites to be validated before execution evidence can be trusted;
- include applicable TestContext/preflight expectations for high-risk or focused evidence;
- context failures (`INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL`, `PREREQUISITE_FAIL`) are not planned product failures and must not be used to justify implementation repair.

## Required plan content

- target backlog item and authoritative acceptance;
- compact **Authority map** naming only product headings/decision IDs that constrain the delta;
- current-state repository evidence, including any forward repair required in code delivered by an Implemented slice;
- affected writers/readers/data/integration/UI surfaces;
- ordered work packages/steps;
- risk profile using `DATA AUTH FIN GOV HIST CONC OPS API UI PERF DEPLOY DEP DOC`;
- material design seam/invariant decisions using `codebase-design` when triggered;
- domain decisions/ambiguities using `domain-modeling` when triggered;
- acceptance map: every authoritative AC → implementation steps → planned independent evidence, plus regression evidence protecting any affected terminal behaviour;
- validation profile/target intent and publishing boundaries.

Keep handoffs compact by referencing step IDs, stable AC IDs and evidence seams
instead of repeating authority prose. Before handing a plan to delivery, run
`python scripts/validate_plan_routing.py <plan>` and fail closed if any package
lacks exactly one signature. Delivery passes that same plan and package to the
reasoning router; a manually supplied kind/risk pair is not a substitute for
the plan-bound signature. Prefer registry profiles/focused targets
exposed by `scripts/validate.py` and label unsupported proposals `INFERRED`
until repository preflight validates them. Do not implement code or absorb
adjacent backlog work. Do not modify a terminal plan to absorb new work. Stop
for genuine unresolved product/security/governance/history/document-authority
decisions rather than guessing.
