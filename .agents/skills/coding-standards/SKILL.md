---
name: coding-standards
description: Apply the canonical tockrPlatform repository engineering standards during planning, implementation and review.
---

# tockrPlatform Coding Standards

Use [`docs/technical/coding-standards.md`](../../../docs/technical/coding-standards.md) as the canonical engineering standard. Read only sections triggered by the actual change, but never ignore a material rule because it was not preloaded.

Use [`docs/technical/testing-execution-contract.md`](../../../docs/technical/testing-execution-contract.md) together with `test-execution` whenever test or validation evidence is planned, executed, reviewed or accepted.

## Mandatory priorities

1. Platform contract authority under `docs/architecture/` and `docs/contracts/` wins over implementation convenience.
2. Preserve modular-monolith dependency boundaries and deep-module discipline.
3. Enforce server-side authorization at every protected resource boundary.
4. Preserve historical/governance truth; never fabricate past facts.
5. Keep all authoritative document/template/evidence binaries external.
6. Keep provider/network/email side effects outside database transactions.
7. Use behaviour-first tests and risk-proportionate evidence only after valid test context is established.
8. Resolve repository-supported validation through `scripts/validate.py`; documentation/examples do not create a second command authority.
9. Treat `INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL` and `PREREQUISITE_FAIL` as context failures, not product failures; do not mutate application code because of them.
10. Treat independent review, independent acceptance and exact-candidate CI as separate gates.

Use canonical vocabulary rather than inventing near-synonyms. Report blockers, skipped required evidence, failure class and material deviations explicitly.
