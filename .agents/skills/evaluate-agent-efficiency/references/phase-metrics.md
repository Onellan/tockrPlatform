# Phase Metrics

Use phase metrics to identify **where** an agent configuration spends or saves resources. They supplement, rather than replace, whole-run quality and resource metrics.

## Agent-internal canonical phases

These phases are for paired single-agent evaluations such as `record-agent-eval.ps1` / `record-tester-eval.ps1`. Use only these names so paired reports remain comparable:

| Phase | Boundary |
| --- | --- |
| `baseline_context` | Resolve authority/current repository state, establish worktree baseline, classify risk and route capabilities. |
| `implementation` | Make the approved implementation delta, including normal focused behavior/test loops. |
| `debugging` | Evidence-first diagnosis work that is entered only when the debugging trigger fires. Do not include ordinary RED/GREEN implementation here. |
| `validation` | Pre-handoff self-review plus broad/affected-surface validation after implementation is materially complete. |
| `handoff` | Compact implementation/evaluation reporting after validation. |

Record **exclusive phase deltas**, not overlapping cumulative totals.

## Metrics

For each phase, record only metrics the execution surface actually exposes:

```json
{
  "baseline_context": {
    "input_tokens": 1200,
    "cached_input_tokens": 800,
    "output_tokens": 180,
    "tool_calls": 6,
    "elapsed_seconds": 42.5
  },
  "implementation": {
    "input_tokens": 4200,
    "cached_input_tokens": 2100,
    "output_tokens": 760,
    "tool_calls": 15,
    "elapsed_seconds": 310
  }
}
```

Allowed phase metrics:

- `input_tokens`
- `cached_input_tokens`
- `output_tokens`
- `tool_calls`
- `elapsed_seconds`
- `estimated_cost_usd`

Omit an unavailable value or set it to `null`. Never record an unknown metric as zero.

## Capture rules

- Prefer execution-surface counters captured at phase boundaries.
- If only whole-run metrics exist, record whole-run metrics and leave phase metrics absent; do not estimate token splits from text length.
- If a phase never occurs (for example no debugging), omit it rather than recording synthetic zero metrics.
- Baseline and candidate must use the same phase definitions and measurement source for a paired comparison.
- Phase totals should reconcile sensibly with whole-run totals when both use the same counter semantics. Investigate material mismatches before drawing a conclusion.

## Delivery lifecycle telemetry

Normal backlog delivery uses a **separate** lifecycle-level metric stream. Do not force those records into the agent-internal phases above.

`backlog_delivery` records logical invocations with `scripts/record-delivery-metric.ps1` using:

```text
plan
implement
review
test
repair-implement
repair-review
repair-test
release-gate
completion-gate
```

Each logical invocation uses `started`/`completed` events so quota/session interruptions remain visible. The stream is written to ignored `.codex/agent-evals/delivery-<item>.jsonl` and summarized by `scripts/summarize-delivery-metrics.ps1`.

Delivery records may contain:

- `input_tokens`
- `cached_input_tokens`
- `output_tokens`
- derived `total_tokens` only when exact input + output exist
- `turns`
- `tool_calls`
- `elapsed_seconds`
- `estimated_cost_usd`
- configured reasoning effort
- result, repair cycle and finding counts when known

Delivery telemetry exists to answer **which lifecycle phase consumes resources and how much repairs cost**. Paired agent-evaluation metrics answer **whether one agent configuration is cheaper at equivalent quality**. Keep those questions and schemas separate.

The delivery summarizer reports measurement coverage with every total. It may report `tokens per successfully released backlog item` only when a successful completion gate is recorded and exact total-token telemetry covers every interruption-safe logical invocation. Missing telemetry remains unavailable; never backfill it from transcript size or model assumptions.

## Implementation work-package and slice attribution

Schema-v3 delivery telemetry can additionally identify implementation work units:

```json
{
  "phase": "implement",
  "agent": "backlog_implementer",
  "attempt": 1,
  "reasoning_effort": "high",
  "work": {
    "package": "WP3",
    "slice": null,
    "kind": "migration",
    "scope": "work-package",
    "risk_level": "H",
    "risk_codes": ["DATA", "HIST"]
  }
}
```

Use work-unit attribution only when the **actual execution/counter boundary** matches that planned work unit:

- package-scoped invocation -> `package=<plan ID>`, `slice=null`;
- slice-scoped invocation -> `package=<plan ID>`, `slice=<slice ID>`;
- `kind` is the planner's pre-work class: `routine`, `defect`, `migration`, `authorization`, `ui`, or `other`;
- invocation spanning multiple packages -> leave work-unit attribution unavailable and keep phase-level usage;
- a slice must always identify its parent plan work package;
- work-package/slice/kind fields apply only to `implement` and `repair-implement`;
- risk metadata belongs to that exact attributed implementation unit;
- never infer package/slice token shares from lines changed, file counts, elapsed time, transcript size or percentages;
- never create extra agent calls solely to improve telemetry granularity.

The summary provides both:

1. **phase totals** — authoritative whole-delivery accounting; and
2. **work-package/slice rollups** — attribution views over those same implementation invocations.

Do not add rollup tokens back into delivery totals. Work-package attribution completeness is reported separately because a delivery can have complete whole-delivery token accounting while some implementation invocations remain unattributed to a package.

For reasoning-effort evaluation, a package/slice comparison is usable only when the compared executions have the same work-unit identity/scope, comparable risk/authority/state, and exact resource counters. A whole-plan invocation cannot be retrospectively split to manufacture package-level evidence.

## Pre-work reasoning recommendation

`recommend_implementer_reasoning.py` maps the planner's package `kind` to the paired evaluation scenarios. The PowerShell command with the legacy name is a thin compatibility wrapper:

| Work kind | Paired scenario |
| --- | --- |
| `routine` | Routine bounded behavior change |
| `defect` | Defect correction |
| `migration` | Schema and migration change |
| `authorization` | Authorization/privacy change |
| `ui` | User-visible workflow |
| `other` | no automatic scenario match |

A matching `Adopt medium` gate supports a **recommendation** only. Production `medium` also requires an enabled human-approved rule in `.codex/reasoning-routing-policy.json` citing a locally verifiable gate run.

Current production eligibility is deliberately narrower than evaluation capability:

- `E` and `H` stay high;
- Risk `R` with `DATA AUTH FIN HIST CONC OPS API PERF DEPLOY DEP GOV DOC` stays high;
- eligible Risk `R` packages may recommend medium when their mapped scenario passed an `Adopt medium` gate;
- recommendation without approved policy continues high;
- missing/stale/mismatched evidence fails safe high.

Normal #9 history may report high/medium sample counts and exact-token medians for a matching `kind + risk + risk_codes` signature. That history is descriptive evidence only; it cannot prove quality equivalence or authorize medium.

When an approved medium runtime discovers a higher actual risk before mutation, delivery records the attempt as `Reroute` and starts the same package high. The reroute's exact counters remain part of delivery cost; it is not erased to make medium look cheaper.

## Interpretation

Use agent-internal phase data diagnostically:

- high `baseline_context` tokens -> inspect eager context/skill/reference loading;
- high `implementation` tokens -> inspect slice size, repeated requirement prose or unnecessary tool loops;
- high `debugging` tokens -> inspect diagnosis quality and failed-repair frequency;
- high `validation` tokens -> inspect over-validation or repeated broad checks;
- high `handoff` tokens -> inspect reporting verbosity.

Use delivery lifecycle data to identify whether planning, implementation, review, testing, repair or release orchestration dominates whole-delivery cost before choosing the next optimization target.

Use work-package/slice data to determine which **classes of implementation work** are expensive and whether reasoning-level experiments should be scoped to coherent low-risk packages rather than entire backlog items. Work-unit evidence remains diagnostic until paired quality-equivalent evaluations justify a recommendation and explicit policy approval authorizes routing.

A phase-level or work-unit saving does not compensate for weaker review/test evidence. Quality gates remain primary.
