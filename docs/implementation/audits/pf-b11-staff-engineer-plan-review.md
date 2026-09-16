# PF-B11 Staff Engineer plan review

**Review type:** Read-only planning/design review
**Review status:** **PASS TO PLAN IMPLEMENTATION**
**Review date:** 2026-09-16
**Reviewed Platform SHA:** `e037873e9dfdfa22ce347b5ad371f47900f348e9`
**Reviewed CTRL SHA:** `a8338e7f422febf9a70f21ff13aa725ebfe07241`
**Reviewed IMS SHA:** `284825bdf1cb30aa6ed089eb18e9644982655565`

## Scope

This review validates the technical completeness of the newly authorised
PF-B11 plan and its four sequential Slice plans. It is planning evidence only;
it is not implementation, acceptance or publication evidence.

## Observed gap

Platform already contains canonical shared authority, signed v1 assertions,
transactional outbox events and bounded local projection inbox/checkpoint
support. The current contracts explicitly do not make local projections
authoritative and do not provide a consumer read-authority transport. CTRL and
IMS have bounded Platform client seams and local projection metadata, but both
correctly refuse to begin PD-D5-S01 without a matching Platform contract.

## Staff Engineer checks

| Check | Result | Required plan response |
| --- | --- | --- |
| Authority locality | PASS | Keep shared identity/tenancy/access in Platform; keep product roles, billing and product domains in consumers. |
| Deep module/locality | PASS | Use capability-local domain/store/SQLite/HTTP seams; no generic service or shared database. |
| Snapshot consistency | PASS | Require immutable, checksum-bound, source-cursor-bound bootstrap material before consumer cutover. |
| Incremental ordering | PASS | Require opaque global cursor plus aggregate sequence, explicit gap/expiry and resync behavior. |
| Security boundary | PASS | Require per-consumer signed requests, rotation, timestamp/nonce replay protection, bounded resources and redaction. |
| Failure semantics | PASS | All non-current states fail closed; no stale state may broaden access. |
| Operational safety | PASS | Require limits, retention, cleanup, readiness, audit classification and rollback. |
| Consumer compatibility | PASS | Freeze a separate `platform.read-authority.v1` contract for both `tockrctrl` and `tockrims`; do not infer semantics locally. |
| Delivery evidence | PASS | Require independent review, independent acceptance, exact-candidate local validation and publication before handoff. |

## Design conclusion

PF-B11 is technically necessary and correctly bounded as a forward extension;
it does not reopen terminal PF-B1–B10 plans. S01 must freeze the wire
contract before S02/S03 implementation. S04 must certify the exact candidate
and update the consumer handoff. Any request for synchronous Platform reads,
shared product storage, guessed identity mapping or consumer authority inside
Platform is a stop/go failure, not an implementation detail.
