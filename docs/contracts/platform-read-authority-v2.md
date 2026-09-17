# Tockr Platform read authority v2

This is the superseding machine-to-machine source contract for local CTRL and
IMS projections. `platform.read-authority.v1` remains a terminal event-only
contract. Version 2 adds explicit provenance for rows seeded by an ordered
Platform migration that predates the transactional outbox; it does not
reinterpret any v1 field.

## Version and boundary

Every v2 request and response uses the exact version
`platform.read-authority.v2` in `X-Tockr-Platform-Read-Authority-Version`.
Unknown, missing or malformed versions fail closed. The v2 record extension
is used by the immutable snapshot surface. Incremental changes remain the
committed `platform-events-v1` envelope and retain event provenance only.

Platform remains authoritative for User, Authentication, Organisation,
OrganisationMembership, Workspace, WorkspaceMembership, Product,
OrganisationProductEntitlement and UserProductAssignment. CTRL and IMS retain
local projections, product configuration, product roles, Projects and local
authorization. A consumer must not query Platform synchronously for product
requests or open the Platform SQLite database.

The v2 routes are:

```text
POST /api/v1/read-authority/snapshots
GET  /api/v1/read-authority/snapshots/{snapshotID}/records
GET  /api/v1/read-authority/changes
GET  /api/v1/read-authority/status
```

Machine authentication is unchanged from v1: per-consumer Ed25519 request
signing, configured key IDs with rotation overlap, bounded UTC timestamp skew,
durable nonce replay protection and secret-free diagnostics. Browser cookies,
user assertions and product assertions cannot substitute for machine
authentication.

## Shared record allow-list

The entity and relationship allow-list is unchanged from v1:

| `entity_kind` | `id` | Additional fields | Status values |
| --- | --- | --- | --- |
| `user` | `usr_*` | `status` | `active`, `archived` |
| `organisation` | `org_*` | `name`, `status` | `active`, `archived` |
| `organisation_membership` | `omem_*` | `user_id`, `organisation_id`, `role`, `status` | `active`, `revoked` |
| `workspace` | `wsp_*` | `organisation_id`, `name`, `status` | `active`, `archived` |
| `workspace_membership` | `wmem_*` | `user_id`, `workspace_id`, `role`, `status` | `active`, `revoked` |
| `product` | `product.*` | `status` | `active`, `retired` |
| `organisation_product_entitlement` | `ent_*` | `organisation_id`, `product_key`, `status` | `active`, `revoked` |
| `user_product_assignment` | `upa_*` | `user_id`, `organisation_id`, `product_key`, `status` | `active`, `revoked` |

Roles are Platform membership roles only: `owner`, `admin`, `member` and
`viewer`. Product roles, billing facts, passwords, sessions, private keys,
document bytes and free-form metadata remain outside the contract.

Every v2 record has `entity_kind`, `id`, `status` and exactly one explicit
`provenance_kind`:

### Event provenance

```json
{
  "provenance_kind":"event",
  "source_event_id":"evt_example",
  "source_sequence":1,
  "source_schema_version":1
}
```

Event provenance requires an `evt_*` identity, a positive aggregate sequence
and source event schema version `1`. Migration fields must be absent.

### Migration-seed provenance

```json
{
  "provenance_kind":"migration_seed",
  "migration_version":6,
  "migration_name":"product-catalogue-organisation-entitlements",
  "migration_checksum":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
```

Migration-seed provenance is valid only for rows created by the named,
ordered migration before an outbox source event existed. The version is
positive, the name is non-empty and bounded to 200 characters, and the
checksum is a lowercase or uppercase 64-character SHA-256 hex value. Event
fields must be absent. A migration seed is historical bootstrap evidence, not
an event and not a permission to fabricate an event ID or sequence.

For the current Platform database, the two Product rows inserted by migration
6 use `migration_seed` provenance bound to the recorded migration 6 name and
checksum. Later Product changes use committed `platform-events-v1` events.

Relationship IDs and status transitions remain strict. Retired, revoked and
archived facts are retained explicitly; consumers must not omit them to make
the projection appear current. Unknown fields and entity kinds fail closed.

## Snapshot and cursor semantics

Snapshot metadata is unchanged from v1: it contains the exact contract
version, consumer, opaque source cursor, SHA-256 checksum, creation and expiry
timestamps, deterministic record count/page size and `complete`. Only a
complete snapshot is usable. The canonical entity order is:

```text
user, organisation, organisation_membership, workspace,
workspace_membership, product, organisation_product_entitlement,
user_product_assignment
```

Records are then ordered by ID and relationship IDs. The checksum is SHA-256
over the canonical UTF-8 JSON sequence of all records, without paging
metadata. A finalized snapshot never changes; expiry requires a new bootstrap.

The total snapshot is bounded to 10,000 records, each page to at most 500
records, request bodies to 64 KiB and responses to 4 MiB. Page cursors are
opaque, snapshot-bound and consumer-bound. A consumer must verify the complete
snapshot, checksum and source state before claiming current authority.

Incremental changes retain the v1 event allow-list, positive aggregate
sequence and opaque global cursor. Consumers verify cursor continuity and
per-aggregate order. Gaps, stale state, unavailable source, expiry or checksum
mismatch fail closed and require the documented recovery path.

## Compatibility and rollback

V2 is a new exact contract artifact. A v1 consumer must reject a v2 response;
a v2 consumer must not reinterpret a v1 record as a migration seed. CTRL and
IMS must record v2 support and test both event and migration-seed records
before PD-D5-S01 is promoted to Ready.

The v1 event contract and its payload allow-list are unchanged. No event is
backfilled for the migration 6 Product rows. Rollback disables the v2
read-authority surface or returns an explicit unavailable state; it does not
delete Product rows, rewrite migration history or convert seed provenance into
synthetic events. A future v3 requires another contract artifact and may not
reinterpret v2 provenance fields.
