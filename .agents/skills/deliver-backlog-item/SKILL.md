---
name: deliver-backlog-item
description: Orchestrate one tockrPlatform backlog item through planning, implementation, independent review/testing, repair loops and exact-candidate CI.
---

# Deliver a tockrPlatform Backlog Item

Operate exactly one active item. Treat `docs/implementation/IMPLEMENTED.md` as authoritative terminal status.

A terminal slice remains Implemented even when later active work repairs code that originated there. The repair belongs to the current active item; the terminal plan and acceptance record remain frozen historical evidence. Never refresh, amend or reopen a terminal plan to absorb new work. A deliberate material change to the delivered contract requires new or explicitly superseding product/backlog authority.

Use `test-execution` to govern every validation result passed between phases.

## Routing and package execution

Require one compact `Route: kind=<...>; risk=<R|E|H>[codes]` signature per
ordered work package. For every not-yet-started package, run
`python scripts/recommend_implementer_reasoning.py --plan <plan> --work-package <package> --work-kind <kind> --risk-level <risk> --risk-codes <codes>`.
Recommendation is
separate from authorization; `.codex/reasoning-routing-policy.json` is the
only tracked medium authorization and defaults to high.

E/H work always remains high. Risk R work involving DATA, AUTH, FIN, HIST,
CONC, OPS, API, PERF, DEPLOY, DEP, GOV or DOC remains high. Routine Risk R may
use medium only when a locally verifiable paired-evaluation gate and enabled
human-approved policy rule match the exact kind and risk codes. Repairs default
high. The medium runtime re-checks eligibility before mutation and returns
`routing_escalation=high` without mutation when risk increases.

## Flow

1. Read exact backlog authority and terminal ledger.
2. Planner creates/refreshes one plan for the active item only; terminal plans are never refreshed.
3. Resolve genuine planning blockers before mutation.
4. Implementer executes approved delta and produces implementation evidence, including regression evidence for any affected terminal behaviour.
5. Independent reviewer performs read-only engineering review and rejects unauthorised terminal-history changes.
6. Any R1 finding returns to implementer; review repeats.
7. Independent tester executes the authoritative acceptance ledger using repository-resolved execution.
8. Classify tester/validation findings before routing repair:
   - valid `TEST_FAIL` in product scope → implementation repair;
   - `INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL`, `PREREQUISITE_FAIL` → repair test context/environment and re-run, not product code;
   - `TIMEOUT` → diagnose cause first;
   - `POSTCONDITION_FAIL` → repair the owning validation/build/configuration surface.
9. After candidate-changing repair, repeat every affected review/test gate against the new candidate.
10. Publish candidate only if current authority allows repository writes.
11. Bind release evidence to the exact candidate SHA and require complete green CI.
12. Only then may terminal implementation metadata be updated for the newly completed item without rewriting earlier terminal entries.

Reviewer/tester do not fix their own findings. Implementer conclusions do not bias independent gates. Blocked validation is never equivalent to Pass.

If all packages are authorized high, use one whole-plan high implementer
invocation. If any package is authorized medium, route ordered packages
sequentially and checkpoint each package. Persist compact
`.codex/delivery-state/<item>.json` before and after every phase/package/repair/
release transition; preserve the exact next phase/package/stage, package
statuses, routing decisions, evidence IDs, finding IDs and orphaned
`metric_inflight`. Resume the exact pointer, never replay a completed package,
and never treat an orphaned start as completed.

Tester batches compatible acceptance rows behind stable `E##` evidence IDs. The
implementer owns focused behaviour plus risk-triggered release parity; the
tester owns authoritative acceptance plus acceptance-relevant focused
regression; release CI owns broad repository regression/security/build
certification.

## Publishing

CI inspection is read-only by default. Repository/external mutations require authority from the current user/workflow.
