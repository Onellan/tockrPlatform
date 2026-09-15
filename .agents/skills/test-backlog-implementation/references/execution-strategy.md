# Acceptance Execution Strategy

Load this reference when the tester is **not** on the Routine acceptance fast path, or when more than one distinct evidence seam/batch is required.

The goal is to obtain complete independent acceptance evidence with the fewest non-duplicated executions. **Plan/report by acceptance condition; execute by evidence seam.**

## 1. Batch by stable evidence seam

Group evidence that can be proven by one coherent execution. Common batches include:

- domain/store fixture suite;
- migration/fresh-upgrade suite;
- authorization/HTTP role matrix;
- API/report/export compatibility suite;
- lifecycle/concurrency/rollback suite;
- browser workflow;
- repository regression baseline;
- documentation/worktree truth checks.

Assign `E##` IDs before execution and map each batch to every `AC##` row/subcase it genuinely proves. Do not create one command/run per acceptance row when a shared execution is sufficient.

Split a batch when combining it would obscure failure attribution, require unrelated setup, or make a blocked check hide otherwise executable evidence.

## 2. Order by consequence and cost

Prefer this order, adapting only when prerequisites require otherwise:

1. **Blockers/prerequisites** — missing authority, unusable test fixture/runtime, or a static acceptance violation that makes dependent evidence impossible.
2. **P0/P1-sensitive negatives** — authorization bypass, data loss/corruption, historical/financial integrity, forbidden lifecycle/state transitions, rollback/idempotency failures.
3. **Focused authoritative behavior** — positive and negative cases at the smallest stable seam.
4. **Boundary/edge cases** — unknown/zero/null, limits, cross-scope isolation, compatibility variants.
5. **Specialized expensive surfaces** — migration/upgrade, concurrency, runtime/side effects, performance when selected by the test profile.
6. **Browser/user workflow** — only when `UI` acceptance requires real interaction/rendering evidence that lower seams cannot establish.
7. **Broad regression checks** — repository/package-wide checks required by changed/tested surfaces.
8. **Documentation/worktree truth** — current-state claims, diff cleanliness and final baseline comparison.

This ordering discovers fundamental failures before spending resources on slow broad checks, but it does **not** mean stop at the first failure.

## 3. Build a complete repair batch

After a failure:

- continue independent evidence that is safe, relevant, and not invalidated by the failure;
- mark genuinely dependent evidence `Blocked` with the exact dependency rather than fabricating a result;
- collect enough reproducible findings to give the implementer one coherent repair batch;
- do not continue an expensive downstream workflow when a prerequisite failure makes its result meaningless.

## 4. Browser is a late, conditional seam

Browser capability is available but should not be initialized merely because templates/routes changed.

Use the browser when authoritative acceptance requires user-visible interaction/rendering/accessibility-sensitive workflow evidence, or when lower HTTP/store/template seams cannot prove the condition completely.

If lower seams completely prove the authoritative behavior and no visual/interaction acceptance exists, record that evidence without launching the application/browser.

When browser execution is required, reuse one prepared disposable runtime/session across compatible `AC##` rows where safe instead of restarting per row.
