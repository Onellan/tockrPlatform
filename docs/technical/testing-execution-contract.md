# Platform testing and validation execution contract

AI decides what must be proved; the repository decides how proof is executed.
`scripts/validate.py` and `scripts/validation_registry.py` are the executable
authority.

## Test context

Every result records candidate identity, authority/AC, surface, profile,
registry source and preflight status. A focused selector matching zero tests is
`INVOCATION_FAIL`, never PASS.

## Failure classification

| Class | Meaning | Response |
| --- | --- | --- |
| `TEST_FAIL` | valid evidence failed | investigate and repair owning surface |
| `INVOCATION_FAIL` | target/selector invalid | repair context, not product code |
| `ENV_FAIL` / `TOOL_FAIL` | environment/tool unavailable | repair/provision or record blocked |
| `FIXTURE_FAIL` | setup/data failed | repair fixture/context |
| `PREREQUISITE_FAIL` | required path/state missing | repair prerequisite |
| `TIMEOUT` | valid command exceeded bound | diagnose before attribution |
| `POSTCONDITION_FAIL` | validator postcondition failed | repair owning configuration/surface |

Context failures never justify application-code changes.

## Independence

Implementer evidence proves focused behavior and material risk surfaces.
Reviewer evidence is read-only engineering assessment. Tester evidence is an
independent acceptance ledger. Batch/final certification proves the complete
required local profile for one exact candidate.
