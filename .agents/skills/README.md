# Tockr Platform skill registry and taxonomy

This registry merges the current CTRL and IMS delivery capabilities and adapts
their repository references to Tockr Platform. Product authority remains in
`docs/architecture/` and `docs/contracts/`; skills define how work is done, not
what Platform means.

## Workflow skills

| Skill | Responsibility |
| --- | --- |
| `plan-backlog-item` | One repository-grounded active Slice plan without implementation |
| `implement-backlog-plan` | Execute one approved plan/work package with risk-proportionate evidence |
| `deliver-backlog-item` | Plan → implement → review → test → repair → exact candidate release gate |
| `deliver-slice` | Deliver one authorised Slice through the same independent gates |
| `code-review` | Independent engineering-quality review |
| `test-backlog-implementation` | Independent acceptance ledger and read-only tester verdict |
| `audit-codebase` | Whole-repository audit when explicitly requested |
| `test-execution` | Resolve and classify evidence through the local validator |

## Engineering skills

`codebase-design`, `coding-standards`, `domain-modeling`, `frontend-design`,
`tdd`, `diagnosing-bugs`, `resolving-merge-conflicts`,
`architecture-health-review` and `evaluate-agent-efficiency` are available when
their triggers are present. They are not automatically loaded as duplicate
context for every task.

## Delivery rules

- Principal/Staff engineering standards remain mandatory.
- New behavior uses behaviour-first RED → GREEN evidence; covered refactors
  preserve GREEN.
- Review and acceptance are independent and read-only.
- Lane 1 prepares a Batch, Lane 2 implements sequential Slices, and Lane 3
  validates/certifies the Batch.
- Local `scripts/validate.py` is the validation authority; GitHub Actions is not
  used merely to validate implementation.
- Full repository review/certification occurs at Batch and final boundaries,
  not on every small Slice.
- Risk routing signatures are part of every ordered work package in a plan.
- Terminal work remains historical; forward repairs belong to a current plan.

## Platform-specific boundary

The Platform agents may change Platform setup, contracts, plans and future
Platform runtime only when authorised by the active Slice. They must not migrate
CTRL/IMS data, change CTRL/IMS authentication, move product authority, import
production data or implement billing under PF.
