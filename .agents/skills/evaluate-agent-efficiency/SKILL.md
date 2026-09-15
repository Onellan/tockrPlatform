---
name: evaluate-agent-efficiency
description: Compare tockrPlatform agent or skill configurations with whole-run, phase and work-package resource evidence without accepting quality loss, extra repair loops, missed findings, or weaker validation.
---

# Evaluate Agent Efficiency

Use this skill to compare one deliberate agent/skill configuration change with its baseline. It is an evaluation workflow, not backlog delivery: do not weaken product authority, review/test gates, or production configuration merely to obtain a cheaper result.

Change **one** configuration variable at a time and use the same scenario, repository baseline, authority, acceptance conditions, and quality oracle for baseline and candidate.

Read:

- [scenario-rubric.md](references/scenario-rubric.md) for representative scenarios and quality equivalence;
- [phase-metrics.md](references/phase-metrics.md) when the execution surface exposes attributable counters.

## Normal-delivery baseline

Before selecting a configuration to optimize, use the delivery lifecycle telemetry produced by `backlog_delivery` when available:

```powershell
.\scripts\summarize-delivery-metrics.ps1 -Item "<item>"
```

Normal-delivery telemetry answers where whole-delivery resources are actually spent and how much repair loops add. Work-package/slice attribution can additionally show which classes of implementation work are expensive. It is not itself a paired quality experiment and does not justify changing reasoning effort from one successful delivery.

Use several representative completed deliveries when possible. Prefer optimization targets with consistently high measured token/runtime share and adequate telemetry coverage. Do not rank an `unavailable` phase/package as cheap, and do not compare measured totals with partial/unmeasured totals as if coverage were equal.

The ignored delivery JSONL stream and summary are operational evaluation evidence only. They must not become product authority, acceptance evidence, completion metadata or a prerequisite for a valid release.

## Production work-package reasoning routing

Reasoning-effort recommendation and production authorization are deliberately separate.

`backlog_delivery` calls:

```text
python scripts/recommend_implementer_reasoning.py --plan <plan> --work-package <WP##> --work-kind <routine|defect|migration|authorization|ui|other> --risk-level <R|E|H> --risk-codes <codes>
```

`scripts/recommend-implementer-reasoning.ps1` is retained only as a thin
compatibility wrapper.

The recommender combines:

1. the package's **pre-work** planner signature;
2. successful paired evaluation-gate manifests under `.codex/agent-evals/`;
3. descriptive schema-v3 #9 package history when available; and
4. `.codex/reasoning-routing-policy.json` for explicit human-approved production authorization.

Rules:

- normal delivery history may improve confidence/cost context but **never** authorizes medium by itself;
- a recommendation may say medium only when a successful `Adopt medium` paired gate supports the mapped workload scenario;
- current production routing keeps E/H high;
- Risk R with `DATA AUTH FIN HIST CONC OPS API PERF DEPLOY DEP GOV DOC` is not medium-eligible;
- missing/malformed/stale evidence or policy fails safe to high;
- a medium recommendation without a matching enabled policy rule is non-blocking and continues high;
- never edit the routing policy automatically from evaluation output;
- after explicit human approval adds a valid rule, matching future not-yet-started packages route automatically without per-package approval.

The two runtime configs are the **same conceptual implementer role**:

```text
backlog_implementer         -> high
backlog_implementer_medium  -> medium
```

They must retain the same skills, permissions and downstream quality gates. The medium runtime has an additional fail-safe pre-mutation check: if actual current code raises the package to E/H or exposes a disqualifying surface/authority ambiguity, it returns `routing_escalation=high`; delivery records `Reroute` and re-invokes the same package high. A reroute is not a product repair cycle.

If all packages remain authorized high, delivery keeps the existing one-call whole-plan implementation to avoid optimization overhead. Package-by-package execution begins only when at least one package is actually authorized medium; then each planned package becomes a real routing/telemetry boundary and integrated review/test/release still occur once after implementation is complete.

## Record implementation-agent runs

For planner/implementer-style comparisons, use `scripts/record-agent-eval.ps1`.

Always record configuration/scenario, quality evidence, independent reviewer/tester outcomes where applicable, repair/finding counts when available, repository fingerprint, and only resource metrics the execution surface actually exposes. Unknown resource data is never zero.

A candidate run must link its same-scenario baseline with `-BaselineReport`.

## Reasoning-effort evaluation: backlog_implementer

Production high remains the fallback until a complete gate says otherwise; a successful gate is recommendation evidence, not automatic policy mutation.

Evaluate `high` vs `medium` with paired runs across at least:

1. `Routine bounded behavior change`;
2. one high-risk integrity scenario: `Schema and migration change` **or** `Authorization/privacy change`;
3. `Defect correction`.

Add `User-visible workflow` when UI work is common enough to influence routing.

For every pair, require matching quality `Pass`, reviewer/tester `Pass`, no extra repair/R1/P0-P3 counts, preserved required validation, and at least one comparable efficiency metric.

After the required candidate reports exist, run:

```powershell
.\scripts\compare-agent-evals.ps1 `
  -CandidateReports <routine-medium>,<integrity-medium>,<defect-medium>
```

The default minimum meaningful efficiency gain is 10% per required scenario. The gate returns `Adopt medium`, `Keep high`, or `Inconclusive` and never edits production configuration.

An `Adopt medium` gate may support a future human-approved **Risk R package rule** for its mapped scenarios. It does not mean E/H automatically routes medium, and it does not retroactively change completed packages.

## Record tester runs

For `backlog_tester` comparisons, use `scripts/record-tester-eval.ps1`. It wraps the generic recorder and adds a machine-checkable tester observation.

Record for both `high` and `medium`:

- the tester's final acceptance verdict;
- the exact tested-state fingerprint emitted by `scripts/get-test-state-fingerprint.ps1`;
- the independent `Test: [...]` surface profile;
- canonical acceptance results such as `AC01=Pass` / `AC02=Fail`;
- stable finding signatures that identify severity + authoritative acceptance row + defect meaning;
- stable evidence-coverage signatures identifying which acceptance seams were actually exercised;
- exposed whole-run/phase resource metrics.

`QualityOutcome` describes whether the **evaluation run** was valid, not whether the implementation under test passed acceptance. A correct tester run against a deliberately defective implementation can therefore have:

```text
QualityOutcome = Pass
AcceptanceVerdict = Fail
```

Do not manufacture or rename signatures merely to make baseline/candidate hashes agree. The signatures exist to detect missed findings or reduced evidence coverage.

## Reasoning-effort evaluation: backlog_tester

Production remains `model_reasoning_effort = "high"` until the tester-specific gate says otherwise.

Use identical tested state and authority for each baseline/candidate pair across at least:

1. `Routine acceptance`;
2. `Schema and migration acceptance`;
3. `Authorization/privacy acceptance`;
4. `User-visible acceptance`.

Add `Concurrency/financial acceptance` when those surfaces are representative enough to influence the default.

Across the required set, include **at least one known-failure/defect-detection scenario** where the high baseline returns a non-Pass acceptance verdict or a real P0-P3 finding. A suite containing only healthy implementations is insufficient evidence that `medium` detects defects equally well.

A tester `medium` candidate may replace `high` only when every required pair has:

- evaluation `QualityOutcome = Pass` for both runs;
- the identical tested-state fingerprint;
- the identical independently derived surface profile;
- the identical acceptance verdict and per-`AC##` result set;
- the identical canonical finding-signature set;
- the identical evidence-coverage signature set;
- zero blocked/not-run acceptance rows;
- at least the configured measured efficiency gain.

After recording the candidate reports, run:

```powershell
.\scripts\compare-tester-evals.ps1 `
  -CandidateReports `
    .codex\agent-evals\routine-tester-medium.md, `
    .codex\agent-evals\migration-tester-medium.md, `
    .codex\agent-evals\auth-tester-medium.md, `
    .codex\agent-evals\ui-tester-medium.md
```

Use `-RequireConcurrencyFinancialAcceptance` only when that additional scenario is part of the evaluation policy.

Gate results:

- **Adopt medium**: every required pair is acceptance-equivalent on the identical tested state and meets the efficiency threshold;
- **Keep high**: medium changes acceptance classification, misses/adds a finding, changes evidence coverage, tests a different state, or fails the minimum efficiency gain;
- **Inconclusive**: required scenarios, defect-detection evidence, complete acceptance evidence, or comparable resource metrics are missing.

Neither tester evaluation script edits `.codex/agents/backlog-tester.toml`. `Adopt medium` is evidence for a separate explicit configuration change only.

## Phase metrics

For paired single-agent evaluations, when counters are available, use `-PhaseMetricsPath` and the agent-internal canonical phases in [phase-metrics.md](references/phase-metrics.md). For tester comparisons, `baseline_context` includes authority resolution, tested-state fingerprint, ledger and surface routing; `validation` may contain acceptance execution; `handoff` contains the compact tester verdict. Omit phases that do not map cleanly rather than inventing measurements.

Do not feed normal delivery lifecycle rows (`plan`, `implement`, `review`, `test`, repair phases, release gates) into `-PhaseMetricsPath`; those are a separate whole-delivery observability stream described in the same reference.

Phase metrics explain **where** resource use changed; they never weaken the quality gate.

## Interpretation discipline

Do not prefer a cheaper candidate that:

- misses a required acceptance row or finding;
- changes the independent surface profile without evidence that the baseline was wrong;
- reduces required evidence coverage;
- needs extra repair/re-review/retest cycles for implementation work;
- expands scope or weakens authority;
- passes only easy/healthy scenarios.

If quality is equal but no meaningful measured efficiency gain exists, keep `high`. If efficiency appears better but evidence is incomplete, report `Inconclusive`.
