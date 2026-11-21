# Tockr Platform

Tockr Platform is the future shared authority for Tockr identity, tenancy and
product access. This repository currently contains the architectural foundation,
versioned contracts, delivery workflow and Priority PF implementation programme.

PF-B1 through PF-B9, PF-B10-S01 and its terminal S01-R1 runtime asset repair
are implemented; PF-B10-S02 is terminal and the Platform Foundation programme
is certified at the authorised scope.

## Authority and scope

- `docs/architecture/` defines the frozen Platform ownership and integration
  boundaries.
- `docs/contracts/` defines versioned cross-repository contracts.
- `plan/active/` contains the executable PF Slice plans.
- `.agents/skills/` and `.codex/agents/` define the delivery workflow.
- `scripts/validate.py` is the local validation command authority.

## Tockr suite responsibilities

| Product | Position and domain authority |
| --- | --- |
| **Tockr Platform** | Shared user identity, authentication, Organisation/Workspace membership, product entitlement and user assignment. It does not own product Projects or product-specific roles. |
| **TockrCTRL** | Engineering delivery control: CTRL Projects, Engineering WBS, engineering baseline and actuals, physical progress, performance, forecast, control reviews/actions, and CTRL project roles. |
| **TockrIMS** | Broader project governance: IMS Projects, Framework Pack adoption, governance baselines, lifecycle/gates, cross-discipline controls, evidence/assurance, and IMS project roles. |

Platform identity and product access are shared foundations. Each product still
authorizes access to its own Projects and records; a CTRL project role does not
grant an IMS role, and an IMS project role does not grant a CTRL role. Billing
and payment are outside this programme.

## Local validation

```text
python scripts/validate.py list
python scripts/validate.py run full/local
```

The validator reports `PASS`, `NOT_APPLICABLE` and explicit context failures.
No validation result is inferred from GitHub Actions.
