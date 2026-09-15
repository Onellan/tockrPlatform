---
name: implement-backlog-plan
description: Execute one approved tockrPlatform implementation plan or delegated work package with risk-proportionate evidence.
---

# Implement a tockrPlatform Plan

Implement only the approved active delta and preserve unrelated work. Read `docs/implementation/IMPLEMENTED.md` before mutation.

An Implemented slice stays Implemented. Its historical plan, acceptance meaning and delivery evidence are not active implementation surfaces and must not be edited, reopened or reinterpreted merely because later work finds a problem in its code.

Code delivered by a terminal slice remains maintainable. When the current active plan identifies a defect, regression, unsafe assumption, stale documentation or maintainability issue in that code, implement the repair as part of the **current active slice**, not as a late amendment to the old slice. Preserve the relevant historical acceptance behaviour with regression evidence. A material change to the delivered product contract requires new or explicitly superseding authority.

## Start

Read the plan Authority map and delegated work package, then verify it against product authority/current code. Classify actual risk independently:

```text
R | E | H
DATA AUTH FIN GOV HIST CONC OPS API UI PERF DEPLOY DEP DOC
```

Load only triggered engineering skills. Use `test-execution` for every test or validator.

The workflow may delegate either one whole approved plan or one exact
work-package/slice with its route signature and completed package IDs. In
package mode, mutate only that package plus the smallest genuine prerequisite;
do not split cohesive work merely for telemetry or medium routing.

The implementer independently reclassifies actual risk before mutation and
loads engineering skills lazily. Use the Routine fast path only for clearly
bounded Risk R work with no material DATA, AUTH, FIN, GOV, HIST, CONC, OPS, API,
PERF, DEPLOY, DEP, DOC or plan-deviation trigger. A UI Risk R package remains
eligible only when the frontend-design `UI:` signature, incumbent-system
inspection and applicable accessibility/responsive/interaction/visual proof
are explicitly covered; otherwise escalate to high before mutation. A medium
runtime escalates as `routing_escalation=high` before mutation when eligibility
changes. Repairs are high by default.

## Execute

- use behaviour-first evidence only after valid TestContext/preflight;
- resolve repository-supported commands through `scripts/validate.py`; do not invent or reuse commands from memory when a registry route exists;
- treat zero matching tests as `INVOCATION_FAIL`, never PASS;
- maintain deep-module/locality rules;
- enforce authorization and historical/document invariants at shared writer seams;
- keep external side effects outside DB transactions;
- do not broaden scope to adjacent debt;
- attribute forward repairs to the current active plan, never to a reopened terminal plan;
- report material plan deviation.

## Failure routing

Before changing product/application code because evidence failed, classify the failure.

- `TEST_FAIL`: validly executed behavioural/security evidence failed; product repair may be appropriate within scope.
- `INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL`, `PREREQUISITE_FAIL`: repair test context/environment/tooling/fixture/prerequisite and re-run; **do not mutate production/application code because of this evidence**.
- `TIMEOUT`: diagnose whether the cause is product, environment or test design before selecting a repair target.
- `POSTCONDITION_FAIL`: repair the owning validator/build/configuration surface; do not automatically treat it as product behaviour failure.

## Evidence

Produce focused implementation evidence plus required local release-parity checks for elevated/high-risk surfaces. For focused Go evidence prefer `python scripts/validate.py go-test ...`; use named profiles for repository-wide gates. For forward repairs to old code, include regression evidence that the applicable terminal acceptance behaviour remains intact. Clean successful output can be compact, but failed/blocked/mixed evidence must retain candidate identity, command provenance and failure class.

The implementer does **not** call product acceptance or release. Independent reviewer and tester own those gates.

Compress clean evidence to stable check/evidence IDs and retain detailed
diagnostics only for failed, blocked, mixed or unexpected results. The
implementer owns focused behaviour and risk-triggered release parity; it does
not call product acceptance or release.
