# PF-B5-S02 independent tester acceptance

Candidate: `1fbdb1a5a82b3e397d166d2cc24516d4cbf03597`

Verdict: **PASS**.

| AC | Independent evidence | Result |
| --- | --- | --- |
| AC01 — assignment lifecycle and ownership | Owner/admin/system authorization, active-member target validation, current-row uniqueness and history-preserving revoke/regrant tests | PASS |
| AC02 — one effective-access predicate | A member with all required active predicates is allowed; the returned contract contains only shared identity and scope | PASS |
| AC03 — every missing predicate denies | Missing assignment, missing entitlement, wrong Workspace, non-member, inactive assignment, inactive entitlement and inactive membership cases deny | PASS |
| AC04 — immediate revocation and audit | Assignment and entitlement revocation immediately deny; grant/revoke events and historical rows are asserted | PASS |
| AC05 — concurrency and migration | Concurrent duplicate assignment test and migration profile pass | PASS |
| AC06 — HTTP protection and redaction | CSRF, authorization, safe denial, list scope and response-redaction tests pass | PASS |

Required acceptance evidence was executed against the exact candidate through
the repository profiles: `integration`, `migration`, `unit`, `format`,
`architecture`, `security`, `quality` and `race`; all were **PASS**. The S02
SQLite and HTTP tests also passed under focused race execution. Container build
evidence is **NOT_APPLICABLE** because no authorised Dockerfile exists.

One initial repository-wide race attempt hit the existing MFA setup test's
timing-sensitive TOTP window under race instrumentation. The exact candidate
was unchanged and the complete race profile was rerun to **PASS**; no unrelated
MFA change was introduced.

No later Slice was tested or accepted.
