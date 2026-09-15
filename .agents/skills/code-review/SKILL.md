---
name: code-review
description: Independently review one tockrPlatform implementation candidate for engineering quality before acceptance testing.
---

# tockrPlatform Code Review

Remain read-only. Start from the fixed implementation diff/current surfaces, then load only relevant authority/standards context. Treat `docs/implementation/IMPLEMENTED.md` as authoritative terminal status.

Use `test-execution` when reviewing material implementation/test evidence. Verify current-candidate binding, repository-backed command provenance, applicable preflight and failure classification. A code change made only because an invalid invocation/environment/tool/fixture/prerequisite failed is an R1 unless later valid evidence independently justifies the change. Zero matching tests is never passing evidence.

Review correctness and engineering quality, especially:

- authority/domain drift;
- edits that reopen, rewrite or reinterpret a terminal Implemented plan/acceptance record without explicit superseding product authority;
- forward repairs incorrectly attributed to old terminal history;
- missing regression evidence protecting historical acceptance behaviour;
- test evidence with stale candidate identity, missing command provenance or invalid preflight;
- product-code repair triggered by `INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL` or `PREREQUISITE_FAIL` without subsequent valid product evidence;
- duplicated invariants or shallow abstractions;
- Organisation/Workspace/product-access authorization gaps;
- transaction/history/audit/concurrency defects;
- entitlement/assignment and assertion immutability or versioning violations;
- event, email and external-consumer boundary violations;
- persistence/provider detail leakage;
- brittle/insufficient tests;
- operational/retry/idempotency problems;
- unbounded query/resource behaviour;
- unjustified dependencies.

A material intended change to a delivered terminal contract without new or explicitly superseding product/backlog authority is an R1 finding.

Findings:

- R1: must fix before acceptance testing.
- R2: worthwhile non-blocking improvement.
- Note: optional follow-up.

Reviewer Pass means no R1 remains. It is not product acceptance. Do not fix your own findings.
