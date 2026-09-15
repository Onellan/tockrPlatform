# Scenario Rubric

Use a stable scenario from this set. Do not change target request, authority, acceptance conditions, repository baseline, or quality oracle between baseline and candidate.

## Implementation-agent scenarios

| Scenario | Required quality evidence |
| --- | --- |
| Routine bounded behavior change | Focused behavior passes, implementation remains scoped, reviewer passes, tester passes, and no unnecessary abstraction/dependency appears. |
| Documentation-only plan/update | Relevant validators pass, authority/current-state truth is preserved, and no unsupported scope is introduced. |
| Schema and migration change | Fresh/representative upgrade behavior, constraints/history/rollback integrity, reviewer pass, and tester pass. |
| Authorization/privacy change | Server-side allowed/denied evidence across affected projections passes; reviewer has no open R1; tester passes applicable acceptance rows. |
| User-visible workflow | Focused behavior plus applicable browser evidence, denied/error/empty states, repository checks, reviewer pass, and tester pass. |
| Performance-sensitive change | Correctness passes; cost shape/cardinality/query behavior is inspected; claimed improvements have a reproducible baseline; reviewer/tester pass. |
| Runtime/side-effect change | Applicable timeout/cancellation/retry/idempotency/recovery behavior is proven; reviewer/tester pass. |
| Defect correction | Real reproduction, evidence-first diagnosis where needed, root-cause repair, focused regression, reviewer pass, and tester pass. |

## Tester scenarios

Tester reasoning evaluation compares **acceptance-detection quality**, not implementation quality. Each baseline/candidate pair must use the identical tested-state fingerprint.

| Scenario | Required tester evidence |
| --- | --- |
| Routine acceptance | Canonical ledger and focused CORE evidence for a bounded change; same acceptance verdict/results/findings/evidence coverage. |
| Schema and migration acceptance | Same DATA profile, fresh/upgrade/integrity evidence, acceptance results and findings on the identical state. |
| Authorization/privacy acceptance | Same AUTH profile and allowed/denied/exposure evidence across affected projections on the identical state. |
| User-visible acceptance | Same UI profile and required browser/lower-seam evidence, including relevant error/empty/denied states, on the identical state. |
| Concurrency/financial acceptance | Same FIN/HIST/CONC profile and exact semantic/stale/rollback/reconciliation evidence where those surfaces are representative. |

Across the required tester set, at least one scenario must be a **known-failure/defect-detection case**. The high baseline must return a non-Pass acceptance verdict or at least one real P0-P3 finding. This proves that a cheaper candidate still detects defects rather than merely confirming healthy implementations.

A tester pair is not comparable when:

- tested-state fingerprints differ;
- authority/plan inputs differ;
- the surface profile differs without an established baseline-classification error;
- blocked/not-run acceptance rows remain;
- baseline/candidate use different evidence-signature conventions.

## Paired comparison rules

Run the current configuration first. Run the candidate against the same scenario and record it with `-BaselineReport <baseline markdown report>`.

Use `QualityOutcome = Pass` only when the **evaluation run itself** is valid. For tester evaluation, the implementation under test may legitimately have `AcceptanceVerdict = Fail` while `QualityOutcome = Pass` when the tester correctly finds the seeded/known defect.

Unknown resource data is `Not recorded`, never zero.

For implementer reasoning comparisons, record:

- `repair_cycles`;
- `r1_findings`;
- `p0_p3_findings`;
- reviewer/tester verdicts;
- scope/risk/validation equivalence.

For tester reasoning comparisons, record with `record-tester-eval.ps1`:

- tested-state fingerprint;
- acceptance verdict;
- `Test: [...]` surface profile;
- canonical `AC##=Result` set;
- stable finding signatures;
- stable evidence-coverage signatures.

A candidate that is cheaper but loses required evidence or defect detection is not more efficient.

## Phase comparison

Use the canonical definitions in `phase-metrics.md` when counters are available:

```text
baseline_context
implementation
debugging
validation
handoff
```

The same counter source and phase boundaries must be used for a baseline/candidate pair. Do not estimate missing token data from message length.

For tester evaluation, use only phases that map cleanly: `baseline_context` for target/authority/state/ledger routing, `validation` for acceptance execution, and `handoff` for the verdict. Omit the others unless the execution surface exposes an unambiguous equivalent.

Phase metrics are diagnostic. They can reveal whether routing reduces context, whether a lower reasoning effort increases repeated validation, or whether output compaction reduces handoff cost.

## Implementer reasoning-effort gate

To evaluate `backlog_implementer` `high` vs `medium`, provide paired candidate reports for at least:

1. **Routine bounded behavior change**;
2. **Schema and migration change** or **Authorization/privacy change**;
3. **Defect correction**.

Add **User-visible workflow** when browser-facing work materially affects the default.

The multi-scenario gate defaults to a **10% minimum measured efficiency gain per required scenario**. A scenario must also have matching baseline/candidate quality `Pass`, reviewer/tester `Pass`, no additional repair/R1/P0-P3 counts, and preserved authority/risk/required-validation behavior.

If a supplied scenario fails quality equivalence or the minimum gain, keep `high`. If required scenarios or machine-checkable evidence are missing, the result is `Inconclusive`.

## Tester reasoning-effort gate

To evaluate `backlog_tester` `high` vs `medium`, provide paired candidate reports for:

1. **Routine acceptance**;
2. **Schema and migration acceptance**;
3. **Authorization/privacy acceptance**;
4. **User-visible acceptance**.

Add **Concurrency/financial acceptance** when representative enough to influence the default.

Every pair must use the identical tested-state fingerprint and produce identical surface profile, acceptance verdict/results, finding signatures, and evidence-coverage signatures. Required scenarios must have zero blocked/not-run rows and at least one supplied scenario must test real defect detection.

The tester gate also defaults to a **10% minimum measured efficiency gain per required scenario**. A mismatch in acceptance/finding/evidence output is `Keep high`; incomplete scenario/evidence coverage is `Inconclusive`.

## Example tester recording

```powershell
.\scripts\record-tester-eval.ps1 `
  -Scenario "Authorization/privacy acceptance" `
  -Configuration "tester medium" `
  -QualityOutcome Pass `
  -QualityEvidence "Same role matrix and exposure seams executed as baseline." `
  -ImplementationSummary "Known authorization defect scenario." `
  -AcceptanceVerdict Fail `
  -TestStateFingerprint <64-char-state-sha> `
  -SurfaceProfile "CORE AUTH API" `
  -AcceptanceResults "AC01=Pass","AC02=Fail","AC03=Pass" `
  -FindingSignatures "P1|AC02|cross-workspace read allowed" `
  -EvidenceSignatures "AC01|auth-role-matrix","AC02|cross-workspace-deny","AC03|shared-output-absence" `
  -InputTokens 9400 `
  -OutputTokens 1600 `
  -ToolCalls 18 `
  -BaselineReport .codex\agent-evals\auth-tester-high.md
```

## Example tester gate

```powershell
.\scripts\compare-tester-evals.ps1 `
  -CandidateReports `
    .codex\agent-evals\routine-tester-medium.md, `
    .codex\agent-evals\migration-tester-medium.md, `
    .codex\agent-evals\auth-tester-medium.md, `
    .codex\agent-evals\ui-tester-medium.md
```
