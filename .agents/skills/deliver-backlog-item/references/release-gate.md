# Release Candidate and CI Gate

Load for `next_phase: release-gate` or release-CI failure. The exact published candidate must pass the repository's real CI before terminal `Implemented` metadata.

## Preconditions

Require reviewer Pass/no R1, independent tester Pass/all authoritative AC, revalidated tester state, no unresolved P0-P3 and no known failed locally reproducible release-parity gate.

## Publication and equivalence

Do not broaden Git permissions. If the accepted implementation is unpublished, checkpoint `release-ready/awaiting_publish_authority`, preserve tester/equivalence evidence and stop before terminal metadata.

Bind release evidence to one exact `release_candidate_sha`. Exact tested SHA is direct equivalence. Publication-only identity change may preserve tester Pass only under `repair-resume.md` positive equivalence; otherwise retest the published candidate. Any product/runtime change invalidates prior review/test normally.

## CI inspection ladder

For the exact candidate, inspect the **smallest evidence surface first**:

```text
workflow status
-> jobs summary
-> failed job(s) only
-> step summaries for failed job(s)
-> log for failed job only when diagnostic text is needed
-> extract/retain only failing-step context
```

Rules:

1. find the repository `CI` run for the exact candidate SHA;
2. if workflow is pending, record `awaiting_ci`; do not fetch logs;
3. if workflow succeeds, record required job success without opening logs;
4. if workflow fails, fetch job summaries and identify only failed/cancelled required jobs;
5. fetch step summaries for those failed jobs before any log download;
6. fetch a failed job's full log only when the step summary does not provide enough diagnosis; do **not** fetch successful-job logs;
7. when a job log must be fetched, reason/report from the failed step's nearby diagnostic lines only—do not carry checkout/setup/cache/successful-step output into repair context;
8. one concise failure record is enough when the same root cause produces repeated log lines.

Compact failure state:

```text
ReleaseState:
candidate=<sha>
ci=failure
run=<id/ref>
failed_job=<name>
failed_step=<name>
diagnostic=<rule/error + file/symbol/line when available>
```

Do not treat Wiki/docs workflows as release CI or accept CI from another SHA.

## Failure routing

Classify from the failed-step diagnostic:

- product/runtime/security/build defect -> `repair_source: release-ci`, repair implement -> fresh review/test;
- test/evidence-harness-only -> repair + category-B targeted tester revalidation;
- docs/completion-truth -> correction + category-C DOC validation;
- external GitHub/registry/service/credential outage -> `Blocked`/`awaiting_ci`, not product defect.

Send the repair implementer the compact failure record plus only precise reproduction/evidence needed. Never attach the full CI transcript when the failing-step excerpt is sufficient.

Never disable/bypass/weaken/allow-list/suppress a required gate merely to obtain green CI unless separately authorised as a policy change.

## Success handoff

After exact candidate CI is green, checkpoint compactly:

```text
ReleaseState:
candidate=<sha>
ci=success
run=<id/ref>
failed_job=none
failed_step=none

last_completed_phase=release-gate
next_phase=completion-gate
```

Only then may terminal completion metadata record backlog `Implemented`, ledger entry, historical plan/technical-plan status and required current-state truth.