# PF-B8-S02 independent tester acceptance

Implementation candidate accepted: `13315d0fbb2c3b2163f9b34c4f8449de4cefb735`

Acceptance was executed after the final Workspace role-control repair and is
bound to the exact implementation candidate above. It covers server-rendered
Organisation, Workspace and System Admin surfaces, direct authorization,
CSRF, audit/history disclosure, responsive presentation and the preserved
Platform boundary.

| Acceptance surface | Command/evidence | Result |
| --- | --- | --- |
| Organisation admin shell and actions | `go test -count=1 ./internal/platform/http` — owner/admin rendering and rename flow | PASS |
| Member action visibility and denial | Focused HTTP test — member sees read-only General, Members direct URL is `404`, member mutation is denied | PASS |
| Workspace admin/member/viewer behavior | Focused HTTP test — administrator controls render, viewer receives read-only context, viewer mutation is `404` | PASS |
| System Admin catalogue | Focused HTTP test — System Admin catalogue renders; non-system direct access is `404` | PASS |
| CSRF | Focused HTTP test — missing form CSRF is `403`; valid form mutation succeeds and redirects | PASS |
| Existing authority/read seams | `go test -count=1 ./internal/store ./internal/db/sqlite ./internal/platform/http ./web/...` | PASS |
| Changed-package race validation | `go test -race -count=1 -timeout=10m ./internal/platform/http ./internal/db/sqlite` | PASS; HTTP 414.803s, SQLite 277.342s |
| Static analysis | `go vet ./...` | PASS |
| Templ/Tailwind generation | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/frontend-tools.ps1 generate-runtime` | PASS |
| Repository profiles | Format, frontend, architecture, security and quality profiles | PASS |
| Plan routing | `python scripts/test_plan_routing.py` | PASS |
| Candidate diff hygiene | `git diff --check` | PASS |
| Wide/narrow browser presentation | Signed-in Members and Workspace Admin journeys at 1440px and 375px; semantic snapshots and screenshots passed | PASS |
| Browser console | Final browser session | PASS; 0 errors, 0 warnings |
| Composite `full/local` diagnostic | `python scripts/validate.py run full/local` | FAIL classification retained: race child `TIMEOUT` at fixed 300s; all other children PASS |
| Container profiles | No authorised Dockerfile exists | NOT_APPLICABLE |
| SQLite connection policy | `db.SetMaxOpenConns(1)` remains unchanged; no pool or migration change | PASS |
| Platform boundary | No CTRL/IMS operational screen, product role, billing, production-data import or authority cutover | PASS |

The `full/local` result is not represented as PASS because its repository-wide
race child has a fixed 300-second bound. The equivalent focused race command
passed under the explicit 10-minute bound, so the required S02 concurrency
evidence is independently available and candidate-bound.

Verdict: **PASS** for the PF-B8-S02 acceptance criteria. No required gate is
BLOCKED / NOT RUN.
