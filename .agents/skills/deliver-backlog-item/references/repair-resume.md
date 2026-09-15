# Repair and Tester-State Resume Protocol

Load this reference only when `next_phase: repair-loop`, when recording a tester result, or when revalidating saved tester evidence at `release-gate` / `completion-gate`.

The purpose is to make `implement -> review -> test` repair cycles and completed tester evidence resumable across Codex usage-window interruptions without skipping an independent gate or rerunning still-valid work.

## Repair-loop state

A checkpoint in `next_phase: repair-loop` must also record:

```text
repair_iteration: <integer >= 1>
repair_source: reviewer | tester | release-ci
repair_stage: implement | review | test
open_finding_fingerprints: <stable IDs>
addressed_finding_fingerprints: <stable IDs or none>
latest_repair_evidence: <references or none>
latest_review_outcome: Pass | Fail | Blocked | not-run
latest_test_outcome: Pass | Fail | Blocked | not-run
```

`repair_stage` is the exact next specialist turn inside the loop. Never infer it from prose.

### State transitions

Use these durable transitions and checkpoint **before** attempting the next turn:

```text
reviewer Fail
-> repair_iteration += 1
-> repair_source: reviewer
-> repair_stage: implement
-> next_phase: repair-loop

repair implementer completes
-> addressed_finding_fingerprints: <reported set>
-> repair_stage: review
-> next_phase: repair-loop

repair reviewer Pass
-> latest_review_outcome: Pass
-> repair_stage: test
-> next_phase: repair-loop

repair reviewer Fail
-> repair_source: reviewer
-> open_finding_fingerprints: <current R1 set>
-> repair_stage: implement
-> next_phase: repair-loop

repair tester Fail
-> repair_iteration += 1
-> repair_source: tester
-> open_finding_fingerprints: <current tester set>
-> repair_stage: implement
-> next_phase: repair-loop

release CI implementation failure
-> repair_iteration += 1
-> repair_source: release-ci
-> open_finding_fingerprints: <CI failure IDs>
-> repair_stage: implement
-> next_phase: repair-loop

repair tester Pass
-> clear repair_source/stage/open findings
-> last_completed_phase: test
-> next_phase: release-gate
```

A quota pause preserves `repair_iteration`, `repair_source`, `repair_stage`, all finding/evidence fingerprints and the current repository fingerprint unchanged. On valid resume, continue exactly the recorded `repair_stage`.

Do not rerun the repair implementer if `repair_stage: review`; do not rerun review if `repair_stage: test` and the repository/reviewer evidence remains valid.

## Tester-state record

Every completed `backlog_tester` turn must return its `State: <fingerprint>`. Persist, when available:

```text
tester_state_status: current | partial | stale
tester_state_fingerprint: <sha256>
tester_state_head: <HEAD tested>
tester_state_tracked_diff_sha256: <sha256>
tester_state_untracked_file_count: <integer>
tester_state_plan_path: <plan path>
tester_state_authority_paths: <exact paths used by tester fingerprint>
tester_verdict: Pass | Fail | Blocked
tester_acceptance_summary: <compact AC result set>
tester_evidence_refs: <E##/finding references>
tester_revalidation_scope: none | evidence-only | docs-only | full
```

The tester-state fingerprint is independent acceptance identity, not a replacement for the delivery checkpoint fingerprint.

## Classify later changes before invalidating evidence

The exact fingerprint match is the preferred proof. When it differs after the tester turn, inspect the actual changed paths and semantic delta before deciding how much evidence is stale. Never classify from commit message alone.

Use the narrowest **demonstrably correct** category:

### A. Product/runtime behavior change -> full stale

Examples include production Go, runtime templates/assets, schema/migration, dependency/build selection, runtime configuration, authorization logic, public API/HTTP behavior, persistence queries/writers, Docker/runtime behavior, or any change that can alter an authoritative product outcome.

Result:

```text
tester_state_status: stale
tester_revalidation_scope: full
```

Prior engineering review and tester Pass are stale. Route to `review`, then fresh `test`.

### B. Test/evidence-harness-only change -> partial stale

Use only when every changed path is demonstrably test/evidence tooling or fixture content and no product/runtime, plan, authority meaning, schema, dependency/runtime selection, or acceptance contract changed. Examples include `*_test.go` fixtures, testdata, test-only helpers, or repository validation scripts that do not run in the product.

Result:

```text
tester_state_status: partial
tester_revalidation_scope: evidence-only
```

Preserve the product engineering-review Pass. Run a fresh tester revalidation that reuses the unchanged canonical acceptance ledger, reruns the evidence directly affected by the harness change, and reruns the **acceptance-relevant focused regression** needed to establish that changed evidence is trustworthy. Do not automatically promote this to repository-wide `go test ./...`, `go vet ./...`, scanners or container smoke; use the tester core-validation broad-check criteria when the harness/shared surface genuinely requires wider coverage. Exact-candidate release CI still owns full repository regression/security/container certification.

If a test/harness change changes acceptance meaning rather than only how it is exercised, this category does not apply.

### C. Completion/documentation-only change -> behavioral Pass remains current

Use only when changed content is documentation/current-state/completion metadata and no executable product/evidence code changed. If a tested authority file changed, inspect the semantic change.

A narrow terminal metadata transition may preserve behavioral acceptance when it changes only delivery/current-state facts such as `Delivery status`, completing commit/CI evidence, ledger entry, capability current-state truth, or historical-plan status **without changing required behavior or acceptance meaning**.

Result:

```text
tester_state_status: partial
tester_revalidation_scope: docs-only
```

Preserve behavioral AC evidence. Run the applicable documentation validation and rerun any `DOC` acceptance evidence actually affected. Do not rerun unrelated store/HTTP/browser acceptance.

If required behavior, acceptance text, domain meaning, security/privacy policy, financial meaning, historical/governance semantics, or product boundary changed, this category does not apply; route to the earliest affected phase.

### D. Publication-only identity change -> preserve only with positive equivalence evidence

A commit/push can change `HEAD` and the fingerprint even when it merely packages the exact tested worktree.

Preserve the tester Pass only when the orchestrator can positively prove the published candidate is byte-equivalent to the tested implementation state using the recorded tested `HEAD`, tracked-diff hash, untracked paths/hashes when available, candidate diff/tree evidence, and unchanged plan/authority content.

If equivalence cannot be proven, test the published candidate. Never assume that a commit containing similarly named files equals the tested worktree.

### E. Plan or substantive tested-authority change -> authority stale

If the plan or tested authority changed beyond the narrow completion-metadata case, route to the earliest phase whose meaning is affected. Normally this is `plan`; a narrower `test` rerun is allowed only when unchanged implementation/design plus exact authority delta proves planning remains valid.

### F. Agent-control-only or quota change -> unchanged

`.codex/` / `.agents/` control-only changes, quota pauses, process restarts, or orchestration metadata do not stale application acceptance by themselves. Ordinary worktree-preservation rules still apply.

## Reusing a tester Pass after interruption or release preparation

When a checkpoint has a tester Pass, recompute the tester fingerprint with `scripts/get-test-state-fingerprint.ps1` using the **same** recorded plan path and authority-path set.

- Exact fingerprint match -> Pass remains current; do not retest.
- Mismatch -> classify under A-F above and record the evidence for that classification.
- Unable to prove a narrow category -> default to the safer broader invalidation, never the narrower one.

A tester revalidation under category B or C remains an independent tester activity for the evidence it reruns; the orchestrator cannot declare those checks passed itself.

## Reusing tester Fail findings after quota interruption

When tester `Fail` has been checkpointed with `repair_stage: implement` and no tested-state-relevant change occurred after that tester turn, preserve its finding fingerprints and reproduction evidence across quota interruption. Resume the repair implementer directly; do not rerun the failing tester just to rediscover the same defects.

If the tested state changed before repair begins, revalidate whether those findings still apply. When that cannot be established from current evidence, obtain fresh tester evidence rather than repairing stale findings blindly.

## Release-CI failure routing

A failed release-candidate CI run is not automatically a product acceptance failure. Inspect the failed step/log and classify:

- implementation/runtime/security/build failure caused by the candidate -> `repair_source: release-ci`, send exact CI evidence to implementer, then review and test according to changed surface;
- test/evidence-harness-only failure -> repair the harness, then apply category B revalidation;
- documentation-only failure -> correct docs and apply category C;
- external GitHub/registry/service outage or unavailable protected credential -> `Blocked`/release pending, not an implementation defect.

Never weaken or bypass a required CI gate to obtain green status.

## Independence rules

- The orchestrator may preserve, classify and route tester evidence; it may not reinterpret a `Fail` as `Pass`.
- Implementer evidence never substitutes for the tester fingerprint/verdict.
- Product/runtime implementation changes after a tester Pass require fresh review and testing.
- Tester acceptance must independently prove every authoritative AC, but need not duplicate broad release-compatibility checks whose only purpose is covered by implementer release parity and exact-candidate CI.
- Evidence-only and docs-only changes may use the narrow revalidation rules above only with positive scope evidence.
- Checkpointing saves already-proven gates; it never weakens the review-before-test order.