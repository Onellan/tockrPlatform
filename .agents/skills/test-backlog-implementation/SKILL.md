---
name: test-backlog-implementation
description: Independently verify one tockrPlatform implementation candidate against authoritative acceptance without changing implementation.
---

# Test a tockrPlatform Implementation

Remain read-only. Build the acceptance model independently from authoritative backlog/product requirements plus the exact candidate state. Do not inherit implementer or reviewer expected verdicts or executable command strings.

Use `test-execution` for every acceptance evidence path.

## Acceptance ledger

Create one row per authoritative acceptance condition with stable `AC##` and evidence IDs `E##`. A single validly executed result may prove multiple rows when it genuinely covers them.

Assign stable `E##` IDs per distinct execution/observation and batch compatible
acceptance rows behind one ID when the execution genuinely proves every mapped
row. Use `scripts/evidence_ledger.py` for ledger-shape validation where
applicable. Do not rerun equivalent checks merely for one-command-per-row
output, and never remove unresolved rows or compress failed/blocked/mixed
evidence.

Test only surfaces actually required, for example:

```text
CORE DATA AUTH FIN GOV HIST CONC API UI DOC PERF OPS DEPLOY DEP
```

For each row:

1. decide independently **what** must be proved;
2. resolve **how** through `scripts/validate.py` / the repository validation registry where supported;
3. establish candidate-bound TestContext and applicable preflight;
4. execute through a durable public/module seam;
5. classify the result before assigning the acceptance verdict.

Focused Go selectors must resolve at least one test. Zero matches is `INVOCATION_FAIL`, never PASS.

## Failure semantics

- `TEST_FAIL` from a validly executed behavioural/security proof may fail the affected AC.
- `INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL`, and `PREREQUISITE_FAIL` make the required evidence `BLOCKED / NOT RUN`; they do not prove product failure and never prove Pass.
- `TIMEOUT` must be diagnosed before attributing the cause to product behaviour.
- `POSTCONDITION_FAIL` must be interpreted against the validator/build/configuration surface it owns.

Use disposable data and stable public seams. Verify denied authorization paths, historical immutability, provider/document bindings and negative state transitions when authority/risk requires them.

Pass requires every required AC to have independent passing evidence bound to the exact candidate. Required unavailable evidence is Blocked, never Pass.

Regression ownership is explicit: the implementer owns focused behaviour and
risk-triggered release parity; the tester owns authoritative acceptance and
acceptance-relevant focused regression; exact-candidate release CI owns broad
repository regression/security/build certification. Do not silently transfer a
release gate into tester evidence.

Do not perform a second general code review and do not fix findings. Delivery still requires complete green CI for the exact published candidate after tester Pass.
