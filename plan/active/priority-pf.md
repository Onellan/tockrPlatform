# Priority PF — Tockr Platform Foundation

Status: **Planned / ready for PF-B1 implementation.**

PF prepares Tockr Platform as the shared authority for identity, tenancy and
product access. It does not implement CTRL/IMS operational screens, billing,
production-data import or authority cutover.

## Programme baseline

| Source | Exact SHA |
| --- | --- |
| TockrCTRL current `main` inspected | `47d20f29ad36e59d44b1154ebcde1ed5a99905f6` |
| TockrIMS current `main` inspected | `aa5de6c4672114c35b3f31fdc3177189a78c9b5d` |
| Platform starting `main` | `55301d8103feee7c0c919703ae01457af0980dcd` |

## Batch and Slice index

| Batch | Purpose | Slices | Status |
| --- | --- | --- | --- |
| PF-B1 | Repository, standards and architecture foundation | S01–S03 | S01 Ready; S02/S03 Planned |
| PF-B2 | Identity and authentication | S01–S02 | Planned |
| PF-B3 | Organisation authority | S01–S02 | Planned |
| PF-B4 | Workspace authority | S01–S02 | Planned |
| PF-B5 | Product catalogue and product access | S01–S02 | Planned |
| PF-B6 | Product assertion and consumer contract | S01–S02 | Planned |
| PF-B7 | Events and local projection support | S01–S02 | Planned |
| PF-B8 | Platform administration UI | S01–S02 | Planned |
| PF-B9 | CTRL/IMS reconciliation and migration tooling | S01–S02 | Planned |
| PF-B10 | Security, runtime and final certification | S01–S02 | Planned |

Total: **10 Batches, 21 Slices**. PF-B1 has three foundation Slices because
the current CTRL/IMS SQLite pool policy conflict must be resolved before
runtime implementation; collapsing it would hide an implementation authority
decision.

## Dependency spine

```text
PF-B1-S01 → PF-B1-S02 → PF-B1-S03
                         ├→ PF-B2-S01 → PF-B2-S02
                         ├→ PF-B3-S01 → PF-B3-S02
                         └→ PF-B4-S01 → PF-B4-S02
PF-B2-S02 + PF-B3-S02 + PF-B4-S02 → PF-B5-S01 → PF-B5-S02
PF-B5-S02 → PF-B6-S01 → PF-B6-S02
PF-B5-S02 → PF-B7-S01 → PF-B7-S02
PF-B6-S02 + PF-B7-S02 → PF-B8-S01 → PF-B8-S02
PF-B6-S02 + PF-B7-S02 → PF-B9-S01 → PF-B9-S02
PF-B8-S02 + PF-B9-S02 → PF-B10-S01 → PF-B10-S02
```

The three authority branches after B1-S03 are sequential within each branch;
they are not parallel implementation permission. Lane 1 must prepare every
Batch, Lane 2 implements one Slice at a time, and Lane 3 certifies the Batch.

## Slice plans

| Slice | Plan |
| --- | --- |
| PF-B1-S01 | [repository, standards, agents and validation](pf-b1-s01-repository-foundation.md) |
| PF-B1-S02 | [ownership and shared contracts](pf-b1-s02-ownership-contracts.md) |
| PF-B1-S03 | [runtime, persistence and presentation foundation](pf-b1-s03-runtime-foundation.md) |
| PF-B2-S01 | [User and authentication](pf-b2-s01-user-authentication.md) |
| PF-B2-S02 | [sessions, MFA, recovery and revocation](pf-b2-s02-sessions-security.md) |
| PF-B3-S01 | [Organisation authority](pf-b3-s01-organisation-authority.md) |
| PF-B3-S02 | [Organisation administration seams](pf-b3-s02-organisation-administration.md) |
| PF-B4-S01 | [Workspace authority](pf-b4-s01-workspace-authority.md) |
| PF-B4-S02 | [Workspace access and scope guard](pf-b4-s02-workspace-access.md) |
| PF-B5-S01 | [Product catalogue and entitlements](pf-b5-s01-product-catalogue.md) |
| PF-B5-S02 | [User assignment and effective access](pf-b5-s02-product-access.md) |
| PF-B6-S01 | [Assertion signing and verification contract](pf-b6-s01-assertions.md) |
| PF-B6-S02 | [Consumer handoff and compatibility](pf-b6-s02-consumer-contract.md) |
| PF-B7-S01 | [Platform events and outbox](pf-b7-s01-events.md) |
| PF-B7-S02 | [Projection inbox and reconciliation](pf-b7-s02-projections.md) |
| PF-B8-S01 | [Layouts, selectors and launcher](pf-b8-s01-platform-shell.md) |
| PF-B8-S02 | [Administration UI](pf-b8-s02-administration-ui.md) |
| PF-B9-S01 | [Reconciliation inventory and mapping](pf-b9-s01-reconciliation.md) |
| PF-B9-S02 | [Dry-run/import and rollback tooling](pf-b9-s02-migration-tooling.md) |
| PF-B10-S01 | [Security and hardened runtime](pf-b10-s01-runtime-hardening.md) |
| PF-B10-S02 | [Final local certification](pf-b10-s02-final-certification.md) |

## Programme stop conditions

Stop PF when a source semantic conflict is unresolved, a product boundary is
ambiguous, an identity mapping would guess, an acceptance condition cannot be
proved locally, or a proposed change would require CTRL/IMS migration or
authority cutover outside this programme.
