# Integrity and Compatibility Planning

Read this reference when the backlog item affects persisted data, legacy records, lifecycle/audit history, authorization/privacy, concurrency, nullable values, or multiple material readers/writers.

## Historical truth and migration

Never manufacture historical lifecycle events, approvals, transitions, reasons, actors, timestamps, classifications, rates, currencies, or prior states from a legacy current state. Preserve known facts exactly and leave unknown facts unknown.

For each affected persisted record, specify fresh-schema and upgrade behaviour, DDL constraints/indexes, migration order, legacy-row mapping, backward compatibility, and any material rollback implication. A migration may map legacy current state to a canonical current state but must not invent how that state was reached. Treat a change to financial, contractual, approval, audit, or governance meaning as a product blocker unless authority settles it.

## Integrity and security

For lifecycle work, name the transaction boundary, permitted state transitions, immutable audit/history facts, and stale-write/idempotency rule. Put shared invariants behind a seam crossed by every material writer.

For authorization or privacy, specify both the permitted actor/action and every server-side read/write/projection boundary that enforces it. UI hiding is not enforcement. Preserve the distinction among `NULL`/unknown, zero, and positive values; do not weaken an explicit required field because a related value is nullable.

## Writer and reader coverage

Enumerate material copy, seed, import, timer, API, report, export, wizard, and edit paths, or identify the shared seam that covers them. The acceptance evidence must prove negative cases: forbidden states/transitions, denied access, rollback or stale writes where relevant, and migration preservation of legacy facts.
