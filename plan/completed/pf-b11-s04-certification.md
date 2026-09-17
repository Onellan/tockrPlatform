# PF-B11-S04 — Security, operability and consumer-readiness certification

**Status:** Implemented / terminal
**Priority:** PF — Platform consumer read-authority extension
**Batch:** PF-B11
**Depends on:** PF-B11-S01, S02 and S03 terminal; current CTRL and IMS plan
reviews complete

## Objective

Independently certify that the complete PF-B11 candidate meets the strict
Platform `platform.read-authority.v2` contract, preserves terminal v1 and
`platform-events-v1` semantics, and is safe to hand to CTRL and IMS for their
separate PD-D5-S01 implementation plans.

## Required review scope

The Staff Engineer and independent tester must review the exact candidate for:

- contract completeness and version compatibility;
- canonical identity, relationship, access and event/migration-seed provenance
  invariants;
- snapshot atomicity, checksum and cursor continuity;
- key rotation, nonce replay protection, authorization and redaction;
- bounded resource behavior, rate limits, retention and cleanup;
- migration fresh/upgrade/reopen behavior;
- stale/gap/blocked/unavailable/resync fail-closed behavior;
- no synchronous per-request consumer dependency;
- no CTRL/IMS product roles, billing, document bytes or database leakage; and
- rollback, readiness, audit and publication evidence.

## Ordered work

### WP01 - Candidate reconciliation

Reconcile every PF-B11 Slice acceptance row, design decision, review finding,
dependency and consumer handoff reference against one candidate.

Route: kind=other; risk=H[GOV,HIST,DOC,DEP]

### WP02 - Complete validation

Run the complete applicable local validation profiles and focused
cross-contract/security/concurrency/resource evidence; classify every result
using the testing execution contract.

Route: kind=other; risk=H[AUTH,DATA,CONC,OPS,API,PERF,DEPLOY]

### WP03 - Independent gates

Perform independent Staff Engineer review and independent tester acceptance,
repair findings sequentially, rerun affected gates and record the exact final
candidate and publication boundary.

Route: kind=other; risk=H[AUTH,DATA,GOV,HIST,DOC,DEPLOY]

### WP04 - Publication and handoff

Update Platform planning/status ledgers and publish the terminal candidate;
prepare the exact contract version/SHA handoff for CTRL and IMS without
promoting their consumer plans automatically.

Route: kind=other; risk=H[GOV,DOC,DEPLOY,DEP]

## Acceptance criteria

- **S04-AC01:** every B11-AC01–B11-AC07 and S01–S03 acceptance row is bound
  to the final candidate and independently evidenced.
- **S04-AC02:** no required security, integrity, migration, concurrency,
  operability, compatibility or publication gate is misclassified as PASS
  when it is blocked or not run.
- **S04-AC03:** the exact published Platform `main` SHA, contract version,
  plan paths, validation evidence and CTRL/IMS handoff are reconciled.
- **S04-AC04:** the final record explicitly states that CTRL and IMS must still
  implement and independently accept PD-D5-S01 in their repositories.

## Evidence and validation intent

Use the complete repository-supported `full/local` profile where its
prerequisites are valid, plus the focused contract, security, migration,
integration, race and architecture evidence required by S01–S03. Container
profiles are required only if the final candidate changes an authorised
container/runtime surface.

### TestContext and preflight

```text
candidate=<exact final PF-B11 certification candidate>
authority=PF-B11-S04-AC01..AC04 and B11-AC01..AC07
surface=AUTH|DATA|CONC|OPS|API|GOV|HIST|DOC|DEPLOY
profile=full/local plus required focused profiles from S01-S03
command_source=validation_registry.py and validate.py
preflight=PASS required before interpreting any behavioral result
```

Every PASS must identify the exact candidate. Required but unavailable
evidence is recorded as `BLOCKED / NOT RUN`; it is never silently omitted from
the certification ledger.

## Stop/go and rollback

Stop and record `BLOCKED / NOT RUN` for any missing independent gate,
unresolved cross-repository contract mismatch, or unavailable required
validation context. Do not mark B11 terminal merely because the API responds
or because a plan was written.

## Completion

PF-B11-S04 is **PASS / terminal** at certification candidate
`6da51a24b281549e5c8084f6c109bd860560154d`. S01, S01-R1, S02 and S03 are
independently reviewed, tester-accepted and exact-candidate validated. The
complete local profile passed every applicable document, security, migration,
unit, integration, frontend, quality and race child. Its AMD64 and ARM64
Docker children are recorded as `ENV_FAIL / BLOCKED / NOT RUN` because Docker
Desktop is unavailable; they are not required because PF-B11 changed no
Dockerfile, image packaging, Compose, generated runtime asset or container
deployment surface.

The engineering review is recorded in
[`docs/implementation/audits/pf-b11-s04-engineering-review.md`](../../docs/implementation/audits/pf-b11-s04-engineering-review.md)
and independent tester acceptance is recorded in
[`docs/implementation/audits/pf-b11-s04-tester-acceptance.md`](../../docs/implementation/audits/pf-b11-s04-tester-acceptance.md).
PF-B11 Batch certification passed and publication is authorized. The published
handoff makes CTRL/IMS PD-D5-S01 eligible for re-evaluation only; it does not
implement either consumer or authorize cutover.
