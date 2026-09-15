# PF-B2-S01 reconciliation evidence

Candidate: `3d283961b8b6f88bd301712555587c2c556a857f`

Plan: [`plan/completed/pf-b2-s01-user-authentication.md`](../../../plan/completed/pf-b2-s01-user-authentication.md)

| Acceptance surface | Evidence | Result |
| --- | --- | --- |
| Stable Platform user identity and active/inactive lifecycle | `go test ./internal/db/sqlite` | PASS |
| Password hashing and unknown-user fixed-cost verification | `go test ./internal/auth`; HTTP denied-login matrix | PASS |
| Secure login/logout and server-side protected route | `go test ./internal/platform/http` | PASS |
| CSRF, secure HttpOnly SameSite cookies, safe headers and bounded forms | HTTP auth tests; architecture/security profiles | PASS |
| Login backoff/rate limiting | HTTP rate-limit test | PASS |
| Security audit without credential disclosure | SQLite audit assertions; protected projection test | PASS |
| Fresh/reopen/divergent migration ledger | SQLite migration tests | PASS |
| One SQLite connection policy | SQLite connection-stat assertion | PASS |
| Independent engineering review | Read-only review of exact candidate | PASS; no R1 findings |
| Independent tester acceptance | Separate acceptance run on exact candidate | PASS |
| Full exact-candidate local profile | `python scripts/validate.py run full/local` | PASS; container profiles NOT_APPLICABLE without Dockerfile |

No CTRL/IMS code, database, identity, migration or authority was changed.
