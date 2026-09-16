# PF-B7-S02 independent tester acceptance

Implementation candidate accepted: `07c2b23ac35645b809fea3b1fc87042932f111b9`

| Acceptance surface | Command/evidence | Result |
| --- | --- | --- |
| Idempotent ordered inbox | `go test -count=1 ./internal/db/sqlite -run 'TestProjectionInbox'` | PASS |
| Duplicate and conflict behavior | Concurrent duplicate ingestion and conflicting identity test | PASS |
| Gap/replay/reconciliation | Out-of-order sequence, bounded promotion, apply and current-state tests | PASS |
| Unknown version retention | Schema-version-2 event retained as blocked and not applicable | PASS |
| Unavailable/recovery state | Explicit unavailable marker and evidence-based resume test | PASS |
| Migration safety | Fresh, upgrade, reopen and divergence migration suite | PASS |
| Changed package acceptance | `go test -count=1 ./internal/db/sqlite` | PASS |
| Changed package race | `go test -race -count=1 ./internal/db/sqlite` | PASS; 217.591s |
| Exact candidate repository race | `go test -race -count=1 -timeout=10m ./...` | PASS; HTTP 321.030s |
| Repository profiles | Format, architecture, security, migration, frontend, quality, unit, integration | PASS |
| Plan routing | `python scripts/test_plan_routing.py` | PASS |
| Container profiles | No authorised Dockerfile exists | NOT_APPLICABLE |

The repository composite `full/local` race child has a fixed 300-second bound,
while the equivalent exact-candidate race command passes in 321.030 seconds.
That timeout remains recorded as diagnostic context and is not represented as
a passing composite-profile result.

Verdict: **PASS** for the Slice acceptance criteria. Projection state remains
non-authoritative for security-sensitive reads and no CTRL/IMS cutover is
claimed.
