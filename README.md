# Tockr Platform

Tockr Platform is the future shared authority for Tockr identity, tenancy and
product access. This repository currently contains the architectural foundation,
versioned contracts, delivery workflow and Priority PF implementation programme.

The PF-B2 identity/authentication Batch, complete PF-B3 Organisation authority
Batch and PF-B4-S01 are terminally implemented; `PF-B4-S02` is the next
sequential active Slice.

## Authority and scope

- `docs/architecture/` defines the frozen Platform ownership and integration
  boundaries.
- `docs/contracts/` defines versioned cross-repository contracts.
- `plan/active/` contains the executable PF Slice plans.
- `.agents/skills/` and `.codex/agents/` define the delivery workflow.
- `scripts/validate.py` is the local validation command authority.

Platform owns shared identity and access decisions only. TockrCTRL continues to
own CTRL Projects and product roles; TockrIMS continues to own IMS Projects,
governance and product roles. Billing and payment are outside this programme.

## Local validation

```text
python scripts/validate.py list
python scripts/validate.py run full/local
```

The validator reports `PASS`, `NOT_APPLICABLE` and explicit context failures.
No validation result is inferred from GitHub Actions.
