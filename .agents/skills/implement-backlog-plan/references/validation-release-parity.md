# Release-Parity Validation

Use this reference when implementation risk is `E` or `H`, Go source changed, and any of `DATA`, `AUTH`, `HIST`, `CONC`, `OPS`, `DEPLOY`, or `DEP` applies. Also use it when the current CI workflow would run application quality/security checks for the changed paths.

The purpose is to reproduce the repository's real release-quality gates before handoff instead of relying only on feature-focused tests.

## Current-CI rule

Read the current `.github/workflows/ci.yml` before executing parity checks. Treat that file as the executable source for what CI currently requires; do not rely on a stale hard-coded command list in this reference.

Run every required CI quality/security check that is locally reproducible with the current permissions/tooling, including the repository boundary/portability checks, full Go test/vet/module-tidiness checks, and configured Go security scanners when those steps exist in CI.

If `OPS` or `DEPLOY` is material, also reproduce the applicable local container/startup/health smoke behavior from CI when Docker/runtime tooling is available.

Cloud-only actions such as GitHub-hosted secret scanning, registry publication, or checks requiring unavailable credentials are not fabricated locally. Record them explicitly as `publish-time CI required`.

## Evidence rule

- A failing locally reproducible CI gate is an implementation defect/blocker for handoff; fix it before claiming release-ready evidence.
- An unavailable locally reproducible gate is reported as `Blocked/not run`; never call it passed.
- A cloud-only gate remains pending for the delivery release gate and does not become an implementer acceptance verdict.
- Do not rerun equivalent checks merely to create duplicate evidence. One parity execution may satisfy both focused validation and CI-parity evidence.
- For each successful parity gate, retain only `<gate/check> -> PASS` plus a decision-relevant metric when useful. Do not carry successful command stdout, setup/cache/package lists, scanner traversal logs or repeated success lines into handoff/checkpoint context.
- For a failed/mixed parity gate, retain the smallest diagnostic record needed for repair: failing gate, rule/test, file/symbol/line when available and observed error. Never suppress a failing sub-result behind an overall command summary.

The independent tester still owns authoritative product acceptance. This parity gate proves release compatibility, not acceptance.