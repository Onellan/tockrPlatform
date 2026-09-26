# PD-D8-S02 — Platform authority integration certification

**Priority:** PD — Platform Authority & Product Access Integration  
**Status:** Ready / next after terminal matched D8-S01  
**Product:** TockrPlatform cross-repository authority  
**Dependencies:** Platform `cc78557d8048bac6392f837166cb101460f43c98`; terminal CTRL/IMS D8-S01 acceptance and published mains.

## Objective

Certify one published Platform/CTRL/IMS authority model, prove Platform is the only enabled writer for shared-authority facts, preserve local product ownership and history, and define a reversible compatibility retirement path. This plan does not authorize production cutover by itself.

## Required candidate baseline

Freeze exact Platform, CTRL and IMS SHAs; contract/schema versions; build identities; deployed authority modes; machine key IDs; D7 manifest, restore-point and receipt hashes; projection cursors; compatibility flags; migration ledger versions; validation reports; and D8-S01 acceptance links. A candidate mismatch invalidates the run.

## Ordered work packages

### WP-PD8S02-01 — Cross-repository semantic diff

Compare lifecycle/status/role meanings, opaque ID prefixes, assertion claims, entitlement/assignment rules, membership command operations/errors, snapshot/feed states and freshness boundaries. Resolve each difference against the published authority contract and record zero unresolved semantic exceptions.

### WP-PD8S02-02 — Sole-writer and local-reader proof

Inventory every shared-authority writer in Platform, CTRL and IMS. Prove Platform is the only enabled writer, consumer commands are bounded adapters, ordinary product requests read local projections without synchronous Platform calls, and CTRL/IMS-owned project, discipline, programme, control and commercial writers remain local.

### WP-PD8S02-03 — Compatibility usage ledger

Inventory legacy auth paths, local shared-membership writers, compatibility tables/columns, adapters and flags. For each item record owner, call sites, runtime counters/log evidence, rollback dependency, retention or removal disposition. Retain by default until zero use and safe rollback are proven.

### WP-PD8S02-04 — Reversible retirement migration

Remove only approved zero-use compatibility items in additive, reversible steps. Preserve local keys, historical actor references, audit/correlation and read-compatible migrations. Destructive schema or history changes require a separate authority decision.

### WP-PD8S02-05 — Cutover, rollback and outage rehearsal

Repeat D7 rollback and D6 outage/recovery on exact candidates after any retirement change. Prove backup restore, projection resync, command correlation, no dual writer, no ordinary-request synchronous dependency and unchanged product data.

### WP-PD8S02-06 — Terminal cross-repository record

Publish matching Platform/CTRL/IMS implementation, independent review and tester acceptance records with exact candidate SHAs, validation IDs, ancestry, clean mains and reconciled queues. Move plans only after every gate passes.

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

Stop on an unresolved semantic difference, enabled local shared writer, missing runtime usage evidence, incomplete restore/rollback, non-ancestor candidate, validation context failure or destructive migration without separate authority. Work is incomplete while any blocker, bug, unresolved acceptance finding or required evidence row remains open. Repair and rerun affected evidence on the exact candidate, then publish clean `main` in all three repositories before terminal closeout.
