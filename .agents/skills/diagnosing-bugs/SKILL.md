---
name: diagnosing-bugs
description: Diagnose unclear, intermittent, performance-related or failed-repair problems from evidence before changing code.
---

# tockrPlatform Diagnosing Bugs

Use safe/non-production data unless authority explicitly allows otherwise. Never expose secrets, provider credentials or confidential document content.

Before treating any failed command as a product defect, use `test-execution` to establish that the command/target is repository-authoritative or validated, the candidate is current, applicable preflight passed, and the failure class is meaningful.

Workflow:

0. classify the failure as product/test (`TEST_FAIL`) versus invocation, environment, tool, fixture, prerequisite, timeout or postcondition failure;
1. reproduce the valid failure;
2. minimise the reproducer;
3. form a small set of falsifiable hypotheses;
4. gather narrow instrumentation/evidence;
5. identify root cause;
6. create regression evidence;
7. apply the smallest cohesive fix;
8. remove temporary instrumentation;
9. rerun the relevant review/test gates against the new candidate.

Do not modify production/application code for `INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL` or `PREREQUISITE_FAIL`. Repair the test context first and re-run. Diagnose `TIMEOUT` before assigning it to product or environment.

Do not stack speculative edits when the feedback loop remains ambiguous. Provider/integration failures must distinguish authentication/authorization, item/version identity, transport/retry and tockrPlatform state errors rather than collapsing them into one generic failure.
