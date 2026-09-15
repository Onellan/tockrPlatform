# Core Acceptance Validation

Use for every tester run. Keep this reference small; add only the surface references selected by the independent test profile.

## Always

- execute focused evidence that directly proves each authoritative acceptance row;
- use stable evidence IDs (`E01`, `E02`, ...) so one execution can support multiple acceptance rows without rerunning it merely for reporting;
- run `git diff --check`;
- compare final `git status --short --untracked-files=all` with the recorded baseline;
- report a required unavailable check as `Blocked`, never `Pass`.

## Regression ownership

The tester owns **independent product acceptance and acceptance-relevant regression**, not duplicate release-compatibility certification.

When Go source changed, start with the smallest package/seam regression set that independently proves the authoritative ACs and materially affected neighboring behavior. Do **not** automatically run:

```text
go test -timeout=10m ./...
go vet ./...
```

Run repository-wide `go test ./...`, `go vet ./...`, scanners, module-wide checks, container smoke or equivalent broad gates only when at least one is true:

- an authoritative AC explicitly requires repository-wide/build/runtime behavior;
- the independently derived `Test: [...]` profile includes `DEP` or `OPS` and the broad check is the smallest reliable evidence for that surface;
- the changed implementation spans/couples enough packages that focused package/seam evidence cannot reliably bound regression risk;
- focused acceptance evidence exposes a failure/warning that requires broader execution to determine acceptance impact;
- a repair/revalidation changed shared test/build infrastructure such that broader regression is directly necessary to validate the affected evidence;
- the release gate/CI is unavailable **and** the broad check is separately required to complete an authoritative acceptance condition.

Do not run a broad check merely because the implementer already ran it, because Go source exists in the diff, or because CI will eventually run it.

The implementer owns risk-triggered local **release-parity** evidence. The delivery release gate owns the exact published candidate's full repository CI quality/security/container result. Neither substitutes for tester acceptance, and tester acceptance does not need to duplicate those checks unless one of the criteria above makes them acceptance-relevant.

When module/build selection changed, use the dependency/runtime reference and select the minimum module/platform checks needed for independent acceptance; full release certification remains owned by exact-candidate CI.

## Evidence-output economy

Execute required evidence fully; compress only its durable representation.

- A clean successful command/observation is retained as `E## <check/seam> -> PASS` plus a small decision-relevant metric when useful.
- Do not carry successful stdout, package-by-package results, setup/cache output, repeated scanner traversal, browser console noise with no finding, or duplicate success prose into the ledger/handoff/checkpoint.
- A `FAIL`, `BLOCKED`, unexpected warning, mixed result, or flaky result remains expanded with the smallest reproduction/diagnostic record needed for independent triage.
- If one command contains multiple required subcases, verify them before compression; never hide a failing/skipped subcase behind an overall `PASS` label.
- Raw verbose output is an execution source, not cross-agent state. Keep only the evidence ID, check identity, verdict and any diagnostic needed later.

The tester may use an existing executed command/test result for more than one acceptance row when it genuinely proves each row. Do not rerun equivalent evidence simply to create one command per row.