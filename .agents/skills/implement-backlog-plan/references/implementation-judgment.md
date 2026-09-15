# Implementation Judgment

Read this reference when classifying implementation risk, deciding whether the approved plan may be adapted, or performing the final self-review. Keep the active run's output compact; this file contains the detailed decision rules so the workflow skill does not need to repeat them.

## Risk profile

Record risk once using:

```text
Risk: <R|E|H> [<codes>]
```

Bands:

- `R` — Routine: bounded/local change with no material risk dimension below.
- `E` — Elevated: one or more dimensions require targeted negative/boundary evidence.
- `H` — High-impact: corruption, unauthorised disclosure/action, incorrect financial/history state, unrecoverable side effects, unsafe migration or material production failure is plausible.

Codes:

- `DATA` — persisted data, schema or migration;
- `AUTH` — authentication, authorization, privacy or security-sensitive input;
- `FIN` — financial/commercial calculations or meaning;
- `HIST` — historical truth, immutable lifecycle or audit;
- `CONC` — concurrency, stale writes, transaction or rollback integrity;
- `OPS` — external side effects, workers, retry/idempotency or restart recovery;
- `API` — public/API/backward compatibility;
- `UI` — user-visible/accessibility-sensitive workflow;
- `PERF` — query/collection/export path with material growth risk;
- `DEPLOY` — configuration, startup, health, storage, secrets or deployment;
- `DEP` — new/changed dependency or supply-chain surface.

Example:

```text
Risk: E [DATA,AUTH]
```

Do not narrate the matrix. Add/update codes only when repository evidence reveals a material risk not known at baseline. Risk controls evidence depth, not feature scope.

## Plan-deviation decision rights

Treat the approved plan as the intended implementation map while reconciling it with current repository truth.

### Decide autonomously

Choose ordinary local implementation details that do not materially change behavior or architecture: private function shape, variable names, query arrangement, fixture organization, equivalent internal structure.

### Adapt and report

Make the smallest unambiguous technical correction when current code proves the planned shape stale or unnecessarily shallow, for example:

- reuse an existing correct seam instead of creating a duplicate;
- omit a planned pass-through abstraction;
- follow a moved/renamed file or symbol that exposes the same contract;
- perform a small cohesive refactor required to centralize the invariant safely.

Preserve product meaning and acceptance. Report only material deviations.

### Stop for authority

Stop rather than choose when proceeding would change required product behavior, authorization/privacy policy, financial meaning, historical/governance semantics, an authoritative acceptance condition, or a material prerequisite outside approved scope.

### Do not absorb

Do not pull in adjacent technical debt, unrelated bugs, broad architecture cleanup, future backlog behavior or "while here" refactors unless necessary to satisfy the active plan safely.

## Pre-handoff self-review

Inspect the complete diff before broad validation. Confirm:

1. scope matches the approved implementation delta;
2. all affected material writers and readers/projections are covered;
3. risk-relevant denied/error/unknown/zero/boundary paths are handled;
4. no unnecessary layer, dependency, configuration or speculative abstraction was introduced;
5. no existing invariant was weakened;
6. temporary debugging, dead/commented alternatives and accidental TODOs are gone;
7. changed behavior has concrete implementation evidence at an appropriate seam;
8. material plan deviations are explicit;
9. pre-existing unrelated worktree paths remain preserved.

Fix clear in-scope defects found here. This is self-review, not the independent engineering-review or acceptance verdict.