---
name: domain-modeling
description: Preserve canonical Tockr Platform identity, tenancy, ownership, access and historical meaning.
---

# Tockr Platform Domain Modeling

Primary authority:

- `docs/architecture/platform-ownership-and-boundaries.md`
- `docs/contracts/platform-contract-v1.md`
- applicable user-authorised Platform decisions and Slice authority.

Use this skill when a change introduces or changes a product concept, lifecycle, ownership rule, authority rule, historical/governance meaning or relationship.

## Rules

- Reuse canonical terms unless a genuinely distinct concept exists.
- Do not create a second glossary in code or technical docs.
- Distinguish implementation drift from a legitimate new concept.
- Stress new concepts with multiple Organisations, Workspaces, products, inactive access and restricted roles.
- Preserve unknown historical facts as unknown.
- Model entitlement, assignment, membership, assertion and reconciliation as governed concepts rather than generic CRUD where product meaning requires it.

Technical naming may differ only when it does not change product meaning and the mapping remains obvious.
