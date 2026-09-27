# PD-D8-S02 — Platform authority integration certification

**Priority:** PD — Platform Authority & Product Access Integration
**Status:** In progress; WP03 compatibility ledger, WP04 bounded retirement and WP05 rehearsal are terminal; WP06 final cross-repository record remains gated
**Product:** TockrPlatform cross-repository authority
**Dependencies:** Platform `cc78557d8048bac6392f837166cb101460f43c98`; terminal CTRL/IMS D8-S01 acceptance and published mains.

## Objective

Certify one published Platform/CTRL/IMS authority model, prove Platform is the only enabled writer for shared-authority facts, preserve local product ownership and history, and define a reversible compatibility retirement path. This plan does not authorize production cutover by itself.

## Required candidate baseline

Freeze exact Platform, CTRL and IMS SHAs; contract/schema versions; build identities; deployed authority modes; machine key IDs; D7 manifest, restore-point and receipt hashes; projection cursors; compatibility flags; migration ledger versions; validation reports; and D8-S01 acceptance links. A candidate mismatch invalidates the run.

## Ordered work packages

### WP-PD8S02-01 — Cross-repository semantic diff

Compare lifecycle/status/role meanings, opaque ID prefixes, assertion claims, entitlement/assignment rules, membership command operations/errors, snapshot/feed states and freshness boundaries. Resolve each difference against the published authority contract and record zero unresolved semantic exceptions.

**Status:** Terminal. The exact candidate comparison recorded zero unresolved exceptions. Evidence: `docs/implementation/audits/pd-d8-s02-wp01-semantic-comparison-2026-09-26.md` and its structured JSON companion. The remaining D8-S02 work packages are still gated in order.

### WP-PD8S02-02 — Sole-writer and local-reader proof

Inventory every shared-authority writer in Platform, CTRL and IMS. Prove Platform is the only enabled writer, consumer commands are bounded adapters, ordinary product requests read local projections without synchronous Platform calls, and CTRL/IMS-owned project, discipline, programme, control and commercial writers remain local.

### WP-PD8S02-03 — Compatibility usage ledger


**Status:** Terminal for inventory and disposition recording. Evidence: `docs/implementation/audits/pd-d8-s02-compatibility-usage-ledger-2026-09-26.md` and its structured JSON companion. All rows are retained; no compatibility path, writer, table or flag was removed. Runtime zero-use evidence and rollback rehearsal remain prerequisites for any later retirement slice.
Inventory legacy auth paths, local shared-membership writers, compatibility tables/columns, adapters and flags. For each item record owner, call sites, runtime counters/log evidence, rollback dependency, retention or removal disposition. Retain by default until zero use and safe rollback are proven.

### WP-PD8S02-04 — Reversible retirement migration


**Status:** Terminal for the bounded retirement candidate. Evidence: docs/implementation/audits/pd-d8-s02-wp04-retirement-2026-09-26.md and the updated compatibility ledger. Two IMS package-private aliases were removed after a repository-wide zero-reference scan; no database migration, ID, historical reference, audit record or rollback receipt was changed. All externally reachable compatibility paths remain retained.
Remove only approved zero-use compatibility items in additive, reversible steps. Preserve local keys, historical actor references, audit/correlation and read-compatible migrations. Destructive schema or history changes require a separate authority decision.

### WP-PD8S02-05 — Cutover, rollback and outage rehearsal

Repeat D7 rollback and D6 outage/recovery on exact candidates after any retirement change. Prove backup restore, projection resync, command correlation, no dual writer, no ordinary-request synchronous dependency and unchanged product data.

**Status:** Terminal. The disposable exact-candidate rehearsal passed backup/restore, projection resynchronisation, membership-command correlation, Platform outage/recovery, C0→C4 cutover, bounded C3→C2 rollback, sole-writer, local-request and product-fingerprint checks. Evidence: docs/implementation/audits/pd-d8-s02-wp05-cutover-recovery-2026-09-26.md and its structured JSON companion.

### WP-PD8S02-06 — Terminal cross-repository record

Publish matching Platform/CTRL/IMS implementation, independent review and tester acceptance records with exact candidate SHAs, validation IDs, ancestry, clean mains and reconciled queues. Move plans only after every gate passes.

Route: kind=defect; risk=E[GOV,DOC,API]

**Ordered execution steps:**

1. **WP06.1 — Repair package-scoped plan routing validation.** Current-state evidence: `scripts/validate_plan_routing.py::validate_text` applies the all-package route-count check before its optional `package` check, so `--package WP-PD8S02-06` still fails on WP01–WP05, whose terminal status and missing historical routing signatures must remain untouched. Update the selected-package path to require exactly one well-formed signature for the requested declared package while ignoring unrelated package signatures; preserve strict all-package behavior when no package is selected. Add regression cases for the D8-S02 mixed terminal/active shape, missing/duplicate/unknown selected packages, and unchanged strict whole-plan behavior. Adjust any active-plan routing test that treats terminal packages as executable targets so it checks current active work without rewriting terminal history. Keep the CLI and routing contract stable for `recommend_implementer_reasoning.py`, which calls `validate_plan(..., work_package)` and consumes `declared_route`.
2. **WP06.2 — Freeze and reconcile the source candidates.** In Platform, CTRL and IMS, identify the exact commit to certify and verify it is the intended `main` candidate, its ancestry from the published baseline, and a clean worktree/index. Bind the contract/schema and build identities; deployed authority modes and machine key IDs; D7 manifest, restore-point and receipt hashes; projection cursors; compatibility flags; migration ledger versions; validation report IDs; and D8-S01 acceptance links. Treat any missing or inconsistent value as a blocker. Preserve existing dirty changes while reconciling them; line-ending-only diffs are still dirty and cannot be silently discarded or certified as a clean main.
3. **WP06.3 — Record implementation state per repository.** On the exact candidate in each repository, record the delivered D8-S02 scope and link the accepted WP01–WP05 evidence, any bounded forward repairs, required validation IDs and exact source SHA. Keep each record aligned with the other repositories' candidate set and state explicitly that production activation is outside this slice.
4. **WP06.4 — Run independent gates per repository.** Obtain a fresh independent engineering review and tester acceptance for each exact candidate. Each record must identify the same Platform/CTRL/IMS candidate tuple, repository-resolved validation profile/target and result IDs, ancestry and clean published `main`; unresolved findings or candidate drift block progression. Do not infer PASS from prior WP01–WP05 evidence when WP06 changes its candidate.
5. **WP06.5 — Reconcile, publish and close.** Check that all nine repository-level records (implementation, independent review and tester acceptance in each of the three repositories) agree with the candidate tuple and validation IDs; AC-PD8S02-01..05 still point to terminal evidence; AC-PD8S02-06 is fully satisfied; and Platform/CTRL/IMS plans and queues reconcile. Publish each repository's clean `main` only after all required gates pass, then write the terminal cross-repository reconciliation and move/update plan indexes. If any gate, ancestry, clean-tree, provenance, validation or reconciliation check fails, leave WP06 open with its exact blocker and do not publish or mark the slice terminal.

**WP06 acceptance map:** AC-PD8S02-06 → WP06.1–WP06.5, proved by package-scoped route validation and regression coverage; the matching exact-candidate implementation/review/tester records; source ancestry and clean-main evidence; registry-resolved validation IDs; and reconciled Platform/CTRL/IMS queues. AC-PD8S02-01..05 remain supported only by their linked terminal WP01–WP05 evidence; WP06 verifies those references and candidate applicability without reopening those packages.

**Validation intent:** Once the validator change is implemented, resolve and run Platform `format` and the supported package-scoped command `python3 scripts/validate_plan_routing.py plan/pd-d8-s02-platform-authority-certification.md --package WP-PD8S02-06`; resolve CTRL and IMS `ci-architecture` profiles from their respective `scripts/validate.py` registries. The registry has no dedicated profile for `scripts/test_plan_routing.py`; use `INFERRED: python3 scripts/test_plan_routing.py` only after preflight confirms the test target and selector behavior, or add/use a repository-supported focused profile if available by implementation time. Bind every result to the exact post-record candidate SHA. Independent reviewers/testers must resolve additional evidence themselves through the relevant repository registry; no hand-copied test command is acceptance evidence. Require valid preflight/context before interpreting results, and classify invocation, environment, tooling, fixture and prerequisite failures as blocked context rather than product failures. No test command or validation profile alone proves ancestry, clean mains or cross-repository reconciliation; those require explicit source-control and record checks.

**Publish boundary:** The routing defect correction and focused regression evidence must pass before freezing the WP06 implementation candidate. Documentation/evidence publication to all three clean `main` branches and terminal plan/index reconciliation are permitted only after WP06.1–WP06.4 pass on the same frozen tuple. This package does not activate production authority, change deployed modes, perform a production migration, or authorize retirement of remaining compatibility paths.

## Acceptance criteria

| AC | Required outcome | Evidence |
| --- | --- | --- |
| AC-PD8S02-01 | Semantic contract diff has zero unresolved rows. | Signed structured diff and independent review. |
| AC-PD8S02-02 | Platform is sole shared-authority writer; consumers use bounded adapters and local projections. | Writer/call graph inventory and runtime traces. |
| AC-PD8S02-03 | Product-owned authority, local history and data remain unchanged. | Domain writer inventory, migration/reopen and before/after fingerprints. |
| AC-PD8S02-04 | Every retired item has zero-use evidence and tested rollback. | Compatibility ledger, focused regressions and rollback receipt. |
| AC-PD8S02-05 | Cutover, outage, resync and restore remain operable. | Exact-candidate rehearsal matrix. |
| AC-PD8S02-06 | Published refs, plans, ledgers and acceptance records agree. | SHA/ancestry and documentation reconciliation. |

## Stop/go and completion

Stop on an unresolved semantic difference, enabled local shared writer, missing runtime usage evidence, incomplete restore/rollback, non-ancestor candidate, validation context failure or destructive migration without separate authority. Work is incomplete while any blocker, bug, unresolved acceptance finding or required evidence row remains open. Repair and rerun affected evidence on the exact candidate, then publish clean `main` in all three repositories before terminal closeout.\n
