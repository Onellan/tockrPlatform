---
name: audit-codebase
description: Audit the whole Tockr Platform repository against its architecture, security, delivery and UX authority without automatically modifying the codebase.
---

# Audit the whole Platform repository

Read authority in this order: `docs/architecture/`, `docs/contracts/`, active
`plan/`, `docs/technical/`, `architecture.md`, implementation and tests. Run
`python scripts/audit_codebase.py --output <REPORT>` and preserve the structured
report. Until Platform runtime exists, the audit must distinguish foundation
contract evidence from runtime `NOT_APPLICABLE` profiles.

## Domains

Evaluate ownership boundaries, authentication/access contracts, migration and
history rules, package/dependency architecture, delivery workflow, local
validation, operability/portability and the initial Platform UI contract.
Static checks do not replace independent architecture review, acceptance or
browser evidence for future runtime Slices.

## Findings

Every finding has an ID, category, P0–P3 severity, files, violated authority,
evidence, impact, remediation, safe-automation flag and proposed Priority →
Batch → Slice placement. P0 is a security/correctness/data-integrity blocker;
P1 is high-impact; P2 is worthwhile; P3 is optional polish.

Do not fix the repository from the report. Convert approved findings into
bounded forward Slices and deliver them through the independent review/tester
workflow. Do not downgrade or reopen terminal history.
