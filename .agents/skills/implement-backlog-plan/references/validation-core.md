# Core Validation

Read this reference for every implementation. Keep validation focused on the changed behavior and then add only the surface references triggered by the risk profile.

Always:

- rerun the focused tests/checks proving the changed behavior or invariant;
- run `git diff --check`;
- compare final `git status` with the recorded worktree baseline.

When Go source changed:

- `gofmt` changed Go files;
- `go test -timeout=10m ./...`;
- `go vet ./...`.

When Markdown changed:

- run `python scripts/check_docs.py`;
- verify completion/current-state claims are supported by implementation evidence.

## Evidence-output economy

Execution depth does not require output retention depth.

- Run every required command/check in full.
- For a clean `PASS`, retain/report only the check identity plus `PASS` and a small decision-relevant metric when useful, for example `go test ./... -> PASS` or `check_docs -> PASS (104 files)`.
- Do not paste, restate, or carry successful package-by-package output, setup/cache noise, repeated success lines, or unchanged command transcripts into implementation handoff/checkpoint context.
- For `FAIL`, `BLOCKED`, unexpected warning, flaky result, or ambiguous outcome, retain the smallest diagnostic excerpt needed to reproduce/route the problem: command/check, failing case/rule, file/symbol/line when available, and observed error.
- If a verbose command is the only execution interface, summarize its successful result after inspection; verbose stdout is evidence source, not durable cross-agent context.
- Never compress away a failed subtest or mixed result by reporting only the command's overall label.

Do not run unrelated validation categories merely for ceremony. Never claim a skipped or unavailable check passed.