# Platform read-authority v2 compatibility

This matrix governs CTRL and IMS consumption of
`platform.read-authority.v2`. It supersedes the v1 record compatibility
surface without changing the v1 event feed. It does not move product-domain
authority or authorise a shared database.

## Bootstrap state machine

Each consumer stores snapshot metadata, records, checksum, source cursor,
contract version and local state in its own database:

```text
request v2 snapshot
  -> verify machine authentication and exact v2 version
  -> persist only complete, checksum-bound pages
  -> validate relationships and provenance kind
  -> verify source state and cursor
  -> atomically promote local projection to current
```

Partial, expired, corrupt, stale, gap, blocked or unavailable snapshots never
become current. There is no synchronous Platform-request fallback.

## Record compatibility

Consumers must support the unchanged eight-entity allow-list and reject
unknown fields, kinds, relationship IDs, product keys, roles or statuses.
Every v2 record must contain exactly one of:

| Provenance kind | Required fields | Forbidden fields | Meaning |
| --- | --- | --- | --- |
| `event` | `source_event_id`, positive `source_sequence`, `source_schema_version=1` | migration fields | A committed source event |
| `migration_seed` | positive `migration_version`, bounded `migration_name`, SHA-256 `migration_checksum` | event fields | A row created by the named ordered migration before outbox creation |

Consumers must preserve the provenance facts as data, must not synthesize an
event identity for a seed and must not infer a Product creation event from a
retirement event. Seed provenance is not a product role, billing fact or
authorization assertion.

## Change feed compatibility

The change feed remains `platform-events-v1`. Consumers validate event ID,
event type, aggregate kind and ID, positive aggregate sequence, schema
version, payload allow-list and global cursor continuity before applying a
change. Duplicate identical events are idempotent; conflicting duplicates,
gaps and stale source state require explicit recovery.

## Required negative cases

Both consumer suites must reject:

- v1, unknown or missing versions when a v2 response is expected;
- an event record without a valid `evt_*` identity, positive sequence or
  schema version;
- a migration seed with event fields, missing migration identity or malformed
  checksum;
- a synthetic event for either migration 6 Product row;
- unknown entity kinds, forbidden fields, invalid relationships or product
  roles;
- incomplete, expired, checksum-mismatched or cursor-inconsistent snapshots;
- stale, gap, blocked or unavailable source state;
- browser cookies or user assertions used as machine authentication;
- response data containing secrets, billing facts, document bytes or SQL
  diagnostics.

The acceptance evidence must bind to the exact Platform v2 contract version
and published Platform candidate. Missing tools, fixtures or environment are
`BLOCKED / NOT RUN`, never acceptance.
