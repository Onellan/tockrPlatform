# PF-B9-S02 fixture import and rollback runbook

PF-B9-S02 provides a signed, fixture-only rehearsal. It is not a production
import procedure and does not authorize CTRL/IMS source access, Platform
database writes, shadow mode, authentication cutover or read/write cutover.

## Preparation

1. Generate a normalized S01 report from disposable fixture/adapter input.
2. Confirm that the report contains no `blocked`, `ambiguous` or `collision`
   proposal. `BuildManifest` refuses a mixed or unresolved report.
3. Obtain an explicit operator ID, approval reason and actual approval time.
4. Sign the manifest with an Ed25519 private key held outside command output.
   The signed manifest is scoped to `fixture-only`.
5. Verify the signed manifest with the corresponding public key before any
   rehearsal.

The `cmd/platform-manifest` command performs steps 2–5 for JSON files. Key
paths are used instead of command-line key material so secrets are not copied
into shell history or logs.

## Rehearsal properties

The `reconciliation.FixtureImporter` is an in-memory disposable state machine.
It applies the manifest in deterministic order and seals each checkpoint with
an HMAC. A tampered checkpoint fails closed. A bounded run returns `paused`,
and a later run resumes from the exact next index. Reapplying a completed
manifest is `idempotent` and does not create duplicate fixture records.

## Rollback

Rollback verifies the same signed manifest and removes only its fixture
records. It appends a compensating audit entry and preserves the signed
manifest, checkpoint history and apply history. A second rollback is a
no-op. Production data, Platform records and source systems are never touched.

## Stop conditions

Stop immediately if the report is ambiguous/blocked, the signature or
checkpoint is invalid, the scope is not `fixture-only`, the manifest identity
does not match the checkpoint, or any operator proposes to select production
data. Those states are blocked evidence, not successful migration.
