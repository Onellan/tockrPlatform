# PD-D8-S02 WP-PD8S02-05 — Final cutover and recovery rehearsal

**Date:** 2026-09-26  
**Verdict:** PASS / rehearsal complete on disposable state  
**Scope:** Exact published `main` candidates; no production activation or product-data migration.

## Exact candidate freeze

| Component | Exact source SHA |
| --- | --- |
| Platform | `f78e3dfb5a2ca76eabee464424c206889f9a7fea` |
| CTRL | `f3a4e10e48d3f7e9b64bf76a088fb74ae6b5265b` |
| IMS | `d1decaab799d13fdb480fa62498f4c20b6fb7884` |

All three repositories were clean at the start and each SHA matched `origin/main`. The accepted D7 handoff inputs remained bound: manifest `2b831b31cd5686ef91c4ac66824f1f1301254d26876ad7cfcfc0e56084f3d211`, manifest SHA256 `9c2f8adbd7831025137907dcb500626cfa438636afc0f5bd0e07ba4a5de78079`, receipt SHA256 `c5b3a5db9e2866c5fa20bdfa1fc3bade4bfa371cadc130ba8fae57b9ea3f453`, and restore point SHA256 `a4d931d8dca81236c8b67eb0e46d5f0bcb4c956833af838a8b4e76799eca1dd`.

## Rehearsal matrix

| Check | Evidence and result |
| --- | --- |
| Backup and restore | CTRL disposable SQLite `TestBackupRestorePreservesReconciliationSnapshot` and representative backup/reopen coverage: **PASS**. The accepted Platform restore point above was re-bound in both consumer evidence files; no schema, identity or history mutation occurred. |
| Projection resynchronisation | Platform read-authority snapshot paging/resync HTTP tests: **PASS**. CTRL/IMS read-authority persistence tests exercised current cursor, duplicate replay, stale/gap fail-closed state, snapshot re-install and recovery: **PASS**. |
| Membership command correlation | Platform membership-command HTTP matrix (signed actor, idempotent replay, stale version, scope denial and outbox rollback): **PASS**. CTRL/IMS command adapters and durable audit correlation (`correlation=corr-*`) tests: **PASS**. |
| Platform outage and recovery | D6 read-authority outage policy tests drove current → blocked/stale → snapshot recovery; no command was allowed while projection state was uncertain: **PASS**. |
| Cutover | `cmd/platform-cutover` preflight and C0→C1→C2→C3→C4 transitions ran for CTRL and IMS. Every receipt carried the exact candidate SHA, current projection (`cur_22`), zero gaps/pending events, resolved mappings, current command-key identifier and verified restore point: **PASS**. |
| Rollback | Saved C3 state was rolled back to C2 with trigger `projection stale after outage`, a 60-second bounded decision and required second freeze/drain, resync and divergence-repair evidence. Both rollback receipts were emitted and persisted: **PASS**. |
| No dual writer | C3/C4 state is `platform/platform`; rollback target C2 is `platform/local`; no state permits `local/platform`. Existing WP02 writer inventory and guard tests remain green: **PASS**. |
| No ordinary-request synchronous Platform dependency | Consumer request and projection suites use local read state; only explicit Platform session/feed and membership-command adapters call Platform. Existing WP02 call-graph evidence and outage tests remain green: **PASS**. |
| Product data unchanged | CTRL post-import and IMS mapping/reconciliation checks retained before/after product-owned fingerprints; rejected writes and rollback were no-op for product-owned data. Existing D8-S01 exact-candidate fingerprints and current full suites remain green: **PASS**. |

## Exact command evidence

- Platform: `go test ./internal/platform/http -run 'TestMembershipCommandHTTP|TestReadAuthorityHTTP' -count=1 -timeout=15m` — **PASS**; `go test ./internal/db/sqlite -run 'TestReadAuthoritySnapshot' -count=1 -timeout=15m` — **PASS**.
- CTRL: `go test ./internal/db/sqlite -run 'TestBackupRestorePreservesReconciliationSnapshot|TestPlatformReadAuthoritySnapshotIsAtomicExactAndFailClosed|TestPlatformReconciliationMappingInstallIsReceiptBoundAndIdempotent|TestPlatformReconciliationPostImportVerificationIsReadOnlyAndReceiptBound' -count=1 -timeout=15m` — **PASS**; HTTP, client and cutover focused suites — **PASS**.
- IMS: generated the ignored runtime stylesheet with `scripts/frontend-tools.ps1 generate-runtime`, then ran the matching HTTP, client, SQLite and cutover focused suites — **PASS**.
- Disposable CLI logs and state/receipt JSON are retained at `C:\Temp\d8-s02-wp05-20260926` for operator inspection; no private key material is present.

**Acceptance:** AC-PD8S02-05 is satisfied. This rehearsal does not activate production authority or retire the remaining guarded compatibility paths; WP-PD8S02-06 remains the documentation closeout gate.
\n