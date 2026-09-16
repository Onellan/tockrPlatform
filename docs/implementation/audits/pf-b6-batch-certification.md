# PF-B6 Batch certification

Batch: PF-B6 — Product assertion and consumer contract

Accepted Batch candidate: `eabcdf22c00d939fc07d5b1ea8eb69d903ab687a`.

## Terminal Slice order

| Slice | Terminal plan | Accepted candidate | Result |
| --- | --- | --- | --- |
| PF-B6-S01 | [`pf-b6-s01-assertions.md`](../../../plan/completed/pf-b6-s01-assertions.md) | `bf3b134e62155481cc98aad7b3613ccdc94129bd`, closeout `cfe24a25e663147f07f8c1d5d5e8fe1e17246477` | PASS |
| PF-B6-S02 | [`pf-b6-s02-consumer-contract.md`](../../../plan/completed/pf-b6-s02-consumer-contract.md) | `e9de6b100eafd19ad75a4b4c3046e0107cb93f62`, closeout `87ffeee58e3f585ec6e6dc99a0b552214e6d0ae2` | PASS |

PF-B6-S02 was not started until PF-B6-S01 was terminally closed. No later
Slice or Batch was implemented.

## Certification gates

| Gate | Evidence | Result |
| --- | --- | --- |
| Slice implementation | Both PF-B6 Slice plans are terminal in `plan/completed/` | PASS |
| Independent engineering review | Separate review evidence for S01 and S02; S02 has no remaining R1 finding | PASS |
| Independent tester acceptance | Separate tester evidence for S01 and S02 | PASS |
| Exact-candidate validation | `full/local` on `87ffeee58e3f585ec6e6dc99a0b552214e6d0ae2` | PASS |
| Programme reconciliation | Tracker, PF index, README, implementation ledger and audit links agree | PASS |
| Boundary and runtime policy | No CTRL/IMS production code, data, role, migration or cutover; one SQLite connection retained | PASS |
| Container profiles | AMD64 and ARM64 builds | NOT_APPLICABLE; no authorised Dockerfile exists |

PF-B6 delivers the versioned Ed25519 assertion and bounded consumer
compatibility contract. Platform issues only the approved shared identity and
scope claims; consumers retain their product roles and governance. The failure
taxonomy is fail-closed and does not expose assertion contents.

The next dependency-ready queue item is PF-B7-S01. PF-B8 and PF-B9 remain
blocked on PF-B7-S02, and no authority cutover is claimed.
