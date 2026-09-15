# Integrity and Compatibility Implementation

Read this reference when the item affects persisted data, legacy records, lifecycle/audit history, authorization/privacy, concurrency, nullable values, or multiple material readers/writers.

## Preserve facts and compatibility

Keep historical facts factual: never invent transitions, approvers, actors, timestamps, reasons, rates, currencies, classifications, or prior states. Fresh databases and supported upgrades must reach a valid schema; preserve known legacy values and leave unknown history unknown.

Use actual migration order. Encode real constraints, foreign keys, uniqueness/check rules, and indexes. Preserve `NULL`/unknown, zero, and positive-value semantics. Do not rewrite historical economics or approvals from current master data.

## Enforce integrity at every path

Apply every shared invariant through each material writer and reader, including copy, seed, import, timer, API, report, export, and edit paths. Enforce authorization and sensitive-field filtering server-side at all relevant read/write/projection boundaries.

For material lifecycle changes, validate current state in the mutation transaction, reject invalid or duplicate transitions, and keep state, immutable history, and audit facts atomic. Use optimistic concurrency or an equivalent stale-write check where a client overwrite could corrupt governed state. Force and verify rollback when partial writes would be damaging.

Add behavioural regression coverage for allowed, denied, boundary, legacy-upgrade, forbidden-transition, stale-write, and rollback cases that apply to the change.
