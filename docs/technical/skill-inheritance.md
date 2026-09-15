# Skill and agent inheritance record

Source baselines: CTRL `47d20f29ad36e59d44b1154ebcde1ed5a99905f6`; IMS
`aa5de6c4672114c35b3f31fdc3177189a78c9b5d`.

## Merged capabilities

Common workflow and engineering skills are present under `.agents/skills/`:

`architecture-health-review`, `audit-codebase`, `code-review`,
`codebase-design`, `coding-standards`, `deliver-backlog-item`, `deliver-slice`,
`diagnosing-bugs`, `domain-modeling`, `evaluate-agent-efficiency`,
`frontend-design`, `implement-backlog-plan`, `plan-backlog-item`,
`resolving-merge-conflicts`, `tdd`, `test-backlog-implementation` and
`test-execution`.

The `.codex/agents/` set includes planner, high/medium implementer, reviewer,
tester, delivery, slice-delivery and audit roles, plus reasoning routing and
resumable delivery-state contracts.

## Merge policy

- Common skills and agents use the later IMS current-main versions where the
  refs differ; those versions include the central validation execution and
  delivery-contract updates.
- CTRL-only openai skill metadata, implementation/review validation references
  and `evaluate-agent-efficiency` are retained.
- IMS-only `audit-codebase`, `deliver-slice`, `test-execution`, audit agent and
  delivery-state capabilities are retained.
- Product-specific names, paths and authority references were adapted to
  `tockrPlatform`; no product-domain rules were copied as Platform authority.
- The merged set preserves Principal/Staff standards, behaviour-first testing,
  independent review/tester gates, three lanes, local validation authority and
  Batch-level/full certification.

## Explicit exclusions

No obsolete historical skill or runtime implementation was copied. GitHub
Actions is not the Platform implementation-validation authority. The current
Platform programme does not enable billing, CTRL/IMS migration or product
operational screens.
