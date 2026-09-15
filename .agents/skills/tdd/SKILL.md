---
name: tdd
description: Implement tockrPlatform behaviour with behaviour-first evidence through stable public seams.
---

# tockrPlatform TDD

TDD starts only after **valid test context** has been established through `test-execution`.

```text
VALID TEST CONTEXT
        ↓
new behaviour        RED → minimum cohesive change → GREEN
reproducible defect  regression RED → root-cause fix → GREEN
covered refactor     GREEN → refactor → GREEN
prerequisite         direct prerequisite proof
docs/config          real validator/runtime proof; no fake RED
```

An invalid command is not RED. A missing tool is not RED. Fixture/prerequisite failure is not RED. Zero matching tests is not GREEN.

Before interpreting a failing test, resolve the supported command through `scripts/validate.py`, require applicable preflight, and classify the result. Only a valid `TEST_FAIL` can provide behavioural RED. `INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL`, and `PREREQUISITE_FAIL` require test-context repair and re-execution, not application-code mutation.

## Test seams

Prefer HTTP + disposable SQLite, store contract + disposable SQLite, public package interfaces, and external-provider adapters. Use browser tests only when browser/UI behaviour is authoritative.

Test behaviour and invariants, not private helper call sequences. Authorization needs allowed + denied evidence. Persistence/history changes need exact final-state/history assertions. Migration tests protect fresh and supported-upgrade behaviour and unknown/NULL semantics.

Implement vertical behaviour slices. Do not declare acceptance; independent tester owns authoritative acceptance.
