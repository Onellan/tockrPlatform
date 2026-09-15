---
name: test-execution
description: Resolve, preflight, execute and classify repository test/validation evidence without guessing commands or confusing invocation/environment failures with product failures.
---

# Test Execution

Use this skill whenever planning, running, reviewing or accepting repository test/validation evidence.

## Core contract

```text
AI decides WHAT must be proved.
Repository decides HOW the proof is executed.
```

Use `scripts/validate.py` and `scripts/validation_registry.py` as executable command authority. Do not invent or reuse a command from memory when the repository can resolve it.

## Before execution

Establish compact context:

```text
TestContext:
  candidate=<current commit/fingerprint>
  authority=<plan/AC or implementation purpose>
  surface=<CORE|DATA|AUTH|GOV|HIST|CONC|API|UI|DOC|PERF|OPS|DEPLOY|DEP>
  profile=<registry profile or focused-go>
  command_source=<validation_registry:...>
  preflight=<PASS required before behavioural interpretation>
```

Use:

```text
python scripts/validate.py list
python scripts/validate.py describe <profile-or-definition>
python scripts/validate.py run <profile>
python scripts/validate.py go-test --package <package> [--run <pattern>]
```

For focused Go evidence, package and selector validity must be preflighted. Zero matching tests is `INVOCATION_FAIL`, never PASS.

## Failure classification

Interpret failures before editing code:

- `TEST_FAIL` — validly executed behavioural/security evidence failed; product repair may be appropriate.
- `INVOCATION_FAIL` — wrong command/target/selector; repair test context, not product code.
- `ENV_FAIL` — environment cannot execute the proof; repair environment/context.
- `TOOL_FAIL` — required tool unavailable; install/provision or record blocked.
- `FIXTURE_FAIL` — setup/data failed; repair fixture/context.
- `PREREQUISITE_FAIL` — required repository/runtime prerequisite missing; repair prerequisite.
- `TIMEOUT` — valid command exceeded bound; diagnose before attributing cause.
- `POSTCONDITION_FAIL` — validator/build/configuration postcondition failed; repair the owning surface, not automatically application behaviour.

Hard rule: **do not modify production/application code because of an invocation, environment, tool, fixture or prerequisite failure.**

## TDD

RED/GREEN starts only after valid test context:

```text
VALID TEST CONTEXT
        ↓
RED → change → GREEN
```

An invalid invocation is not RED. Missing tools are not RED. Zero matching tests is not GREEN.

## Independence

Planner/implementer/tester may independently decide what evidence is required, but command execution must converge on the same repository authority. Tester must not inherit implementer command strings as proof.

## Acceptance

A result can support authoritative acceptance only when it is candidate-bound, repository-resolved, preflight-valid and actually executed where required. Context failures are Blocked/Not-run, never acceptance FAIL or PASS.

## Full/browser adapters

Complex full-profile, container and browser harnesses may keep specialised orchestration. Invoke them through repository profiles where available and preserve their explicit blocked/platform semantics. Do not copy their internal commands into ad hoc agent prompts.

## Output discipline

Report the evidence target, profile/definition, status, failure class and candidate identity. Do not dump this skill or long command inventories into routine reports.
