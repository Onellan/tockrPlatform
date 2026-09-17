# PF-B11-S03 independent tester acceptance

**Candidate:** `bfc111ad1e7f8add6967e2dbb8b41f8ac2d192ea`
**Decision:** **PASS**
**Authority:** PF-B11-S03 S03-AC01–S03-AC06

The tester independently resolved the S03 acceptance model from the active
Slice plan and v2 contract. The implementation candidate was not modified
during testing.

| Acceptance row | Evidence | Result |
| --- | --- | --- |
| S03-AC01 — active consumer signature, timestamp and unused nonce are required | E01 focused HTTP contract tests; signed requests, browser/session rejection, extreme timestamp rejection | PASS |
| S03-AC02 — overlap rotation works and retired, replayed and cross-consumer credentials fail closed | E01 `TestReadAuthorityHTTPReplayRotationAndCrossConsumerBinding` and `TestReadAuthorityHTTPAcceptsOverlappingKeyAndEnforcesConsumerRateLimit` | PASS |
| S03-AC03 — bounded deterministic snapshots and committed v1 changes preserve provenance and safe versioned responses | E01 snapshot paging/change-feed contract test; E02 integration; E03 migration; E04 security | PASS |
| S03-AC04 — cursor expiry/source gaps require resynchronisation | E01 `TestReadAuthorityHTTPSnapshotPagingChangesAndResync`; `cur_999` returns `resync_required` | PASS |
| S03-AC05 — no browser fallback, per-request product role evaluation or shared database path | E01 browser-auth rejection; E05 architecture review; read-only boundary inspection | PASS |
| S03-AC06 — consumer compatibility, negative/resource/security/concurrency evidence | E01 both `tockrctrl` and `tockrims` fixtures; E02–E08 repository profiles and focused HTTP tests | PASS |

## Evidence ledger

- **E01:** `go test ./internal/platform/http -run '^TestReadAuthorityHTTP' -count=1` — PASS; focused HTTP contract and adversarial tests resolved and executed.
- **E02:** `python scripts/validate.py run integration` — PASS; SQLite and HTTP integration packages passed.
- **E03:** `python scripts/validate.py run migration` — PASS; fresh/upgrade/reopen migration suite passed.
- **E04:** `python scripts/validate.py run security` — PASS; no secret-pattern findings.
- **E05:** `python scripts/validate.py run architecture` — PASS; required boundary/runtime authority files present.
- **E06:** `python scripts/validate.py run format` and `python scripts/validate.py run quality` — PASS; format clean and plan/route integrity passed.
- **E07:** `python scripts/validate.py run unit` — PASS; registered unit packages passed.
- **E08:** `python scripts/validate.py run race` — PASS; `go test -race ./...` passed on the candidate.

One earlier generic-skill `validate.py go-test` attempt returned
`INVOCATION_FAIL` because this repository exposes only `list`, `describe` and
`run`; it was not used as acceptance evidence and did not trigger a product
change.

No required S03 acceptance row is blocked, not run or inferred from an
unavailable environment. Tester acceptance is separate from engineering
review and does not authorize S04 until S03 is terminally reconciled.

