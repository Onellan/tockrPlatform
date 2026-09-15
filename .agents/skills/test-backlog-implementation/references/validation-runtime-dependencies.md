# Runtime, Side-Effect and Dependency Acceptance Validation

Load for `OPS` and/or `DEP` surfaces: workers, webhooks, email, storage, imports, external calls, startup/health, configuration/deployment, containers, or dependency/build-graph changes.

As applicable to authoritative acceptance:

- verify timeout/cancellation behavior;
- verify retry, duplicate-delivery and idempotency semantics;
- verify partial failure and restart/recovery for persisted work spanning process lifetime;
- verify actionable diagnostics without secret leakage;
- run startup/health/storage/container smoke checks at the real runtime boundary;
- when imports, `go.mod`, `go.sum`, build tags or dependency selection changed, run `go mod tidy -diff` (or repository-equivalent no-diff check);
- verify relevant supported platform/build compatibility and use installed dependency/security tooling when required by the changed surface.

Do not install or upgrade tooling during independent acceptance testing. Do not run container/security/dependency checks merely because the repository has those tools; route them from the tested surface.