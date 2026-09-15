---
name: deliver-slice
description: Deliver one authorised tockrPlatform implementation slice through implementation, review, acceptance and validation.
---

# Deliver one slice

Operate exactly one authorised slice and preserve terminal history.

Use `test-execution` for every test or validation result. Select what must be proved from authority and risk; resolve supported execution through `scripts/validate.py`.

## Required sequence

1. Resolve scope, authority, risks and acceptance conditions.
2. Establish valid TestContext before interpreting RED or GREEN.
3. Implement only the authorised delta through stable public seams.
4. Run required named or focused validation through the canonical runner.
5. Classify every failure before choosing a repair target.
6. Send valid `TEST_FAIL` product defects to implementation repair.
7. For `INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL` or `PREREQUISITE_FAIL`, repair the test context or environment and re-run; do not change application code because of that evidence.
8. Diagnose `TIMEOUT` before assigning cause. Route `POSTCONDITION_FAIL` to its owning build, configuration or validation surface.
9. Use separate read-only reviewer and tester gates.
10. Re-run affected evidence after any candidate-changing repair.

## Completion

A slice is ready only when no blocking review finding remains, every required acceptance row has independent passing evidence, and required validation for the final candidate is green. Blocked or not-run evidence is never Pass.
