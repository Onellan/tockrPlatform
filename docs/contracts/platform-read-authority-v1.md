# Tockr Platform read authority v1

This is the versioned, machine-to-machine source contract for local CTRL and
IMS projections. It is separate from the short-lived `platform.v1` assertion
contract and from the at-least-once `platform-events-v1` delivery envelope.
It does not grant product access, move product authority or authorize a shared
database.

The first supported wire version is `platform.read-authority.v1`. Unknown,
missing or malformed versions fail closed. The contract is frozen by
PF-B11-S01; runtime snapshot storage is PF-B11-S02 and the HTTP transport is
PF-B11-S03.

## Boundary and ownership

Platform remains authoritative for User, Authentication, Organisation,
OrganisationMembership, Workspace, WorkspaceMembership, Product,
OrganisationProductEntitlement and UserProductAssignment. CTRL and IMS keep
local projections, product configuration, product roles, Projects and local
authorization. A consumer must make product requests from its local current
projection; it must not synchronously call Platform or open the Platform
SQLite file for each request.

The read authority contains shared facts needed to establish Platform-backed
identity, tenancy and product entitlement. It never contains passwords,
password hashes, session tokens, MFA/recovery material, private keys, browser
cookies, product roles, billing/payment facts, document bytes, mutation
reasons or unbounded free-form metadata.

## Version and response headers

Every request sends:

| Header | Requirement |
| --- | --- |
| `X-Tockr-Platform-Read-Authority-Version` | Exactly `platform.read-authority.v1`; unknown or missing values fail closed. |
| `X-Tockr-Platform-Consumer` | Exactly `tockrctrl` or `tockrims`; it is checked against the configured signing key and product binding. |
| `X-Tockr-Platform-Key-ID` | Deployment-managed active or overlap key identifier. |
| `X-Tockr-Platform-Timestamp` | UTC Unix seconds as an ASCII decimal value. |
| `X-Tockr-Platform-Nonce` | Unique per consumer/key; 1–128 token characters. |
| `X-Tockr-Platform-Signature` | Base64url without padding of the Ed25519 signature over the canonical request. |

Every response includes
`X-Tockr-Platform-Read-Authority-Version: platform.read-authority.v1`.
Responses use JSON UTF-8 and must not include secrets or provider/SQL
diagnostics.

The read-authority route family is limited to:

```text
POST /api/v1/read-authority/snapshots
GET  /api/v1/read-authority/snapshots/{snapshotID}/records
GET  /api/v1/read-authority/changes
GET  /api/v1/read-authority/status
```

These are integration routes, not browser-session routes. A valid browser
cookie, Platform user assertion or product assertion cannot substitute for the
machine signature.

## Machine request authentication

The signed bytes are the UTF-8 concatenation below, with exactly one LF
between fields and no trailing LF:

```text
UPPER(method)
request-target-path
lowercase-hex(SHA-256(raw-request-body))
consumer
key-id
timestamp
nonce
```

The request-target path includes the route and a canonical query component
when present; query keys and values are sorted and percent-encoded by the
route implementation before signing. The body digest is calculated before JSON
parsing. `GET` and `HEAD` requests sign the SHA-256 digest of an empty body.
Header values are trimmed once and then treated as exact values; CR/LF is
invalid.

Platform verifies the Ed25519 signature in constant time against the key ID
and consumer binding. Timestamp skew is at most 300 seconds from Platform's
UTC clock. The nonce is durably reserved for at least the maximum skew plus
the configured request lifetime, keyed by consumer + key ID + nonce. A reused
nonce, unknown/retired key, malformed signature, cross-consumer key, future or
expired timestamp is rejected without revealing which check failed.

Key rotation uses an overlap window: the new key is accepted before the old
key is retired, and a retired key is rejected after the configured retirement
point. Public keys and key IDs are deployment configuration; private keys
never enter Platform source, SQLite, response bodies, audit messages or logs.

## Shared record allow-list

Each record has the common fields `entity_kind`, `id`, `status`,
`source_event_id`, `source_sequence` and `source_schema_version`. IDs are
opaque Platform IDs and are not parsed by consumers. `source_event_id` is an
`evt_*` event identity, `source_sequence` is the positive sequence for its
aggregate, and `source_schema_version` is the integer event schema version.
All three provenance fields are mandatory.

The complete entity and relationship allow-list is:

| `entity_kind` | `id` | Additional fields | Allowed relationship meaning | Status values |
| --- | --- | --- | --- | --- |
| `user` | `usr_*` | `status` | Platform identity only | `active`, `archived` |
| `organisation` | `org_*` | `name`, `status` | Organisation lifecycle | `active`, `archived` |
| `organisation_membership` | `omem_*` | `user_id`, `organisation_id`, `role`, `status` | One User in one Organisation | `active`, `revoked` |
| `workspace` | `wsp_*` | `organisation_id`, `name`, `status` | One Workspace owned by one Organisation | `active`, `archived` |
| `workspace_membership` | `wmem_*` | `user_id`, `workspace_id`, `role`, `status` | One User in one Workspace | `active`, `revoked` |
| `product` | stable `product.*` key | `status` | Catalogue identity only | `active`, `retired` |
| `organisation_product_entitlement` | `ent_*` | `organisation_id`, `product_key`, `status` | Organisation access to a Product | `active`, `revoked` |
| `user_product_assignment` | `upa_*` | `user_id`, `organisation_id`, `product_key`, `status` | User assignment within an Organisation and Product | `active`, `revoked` |

The relationship invariants are strict: referenced IDs must have the matching
prefix and kind; a Workspace has exactly one Organisation; memberships have
one User and one parent; entitlements have one Organisation and Product; and
assignments have one User, Organisation and Product. A revoked/archived/
retired fact is retained as an explicit historical state; it is never omitted
to make a consumer appear current.

Roles in membership records are only Platform membership roles: `owner`,
`admin`, `member` and `viewer`. They are not product roles. The product key
allow-list for this version is `product.tockrctrl` and `product.tockrims`.

## Snapshot bootstrap

The consumer first requests a bounded immutable snapshot:

```http
POST /api/v1/read-authority/snapshots
Content-Type: application/json
```

```json
{"entity_kinds":["user","organisation","organisation_membership","workspace","workspace_membership","product","organisation_product_entitlement","user_product_assignment"],"expires_in_seconds":3600}
```

`entity_kinds` is non-empty, duplicate-free and drawn only from the allow-list.
`expires_in_seconds` is positive and no greater than 86,400. The authenticated
consumer is taken from the request headers, not trusted from this JSON body.

The successful response returns immutable metadata:

```json
{
  "version":"platform.read-authority.v1",
  "snapshot_id":"snap_*",
  "consumer":"tockrctrl",
  "contract_version":"platform.read-authority.v1",
  "source_cursor":"cur_*",
  "checksum_algorithm":"sha256",
  "checksum":"<lowercase hex>",
  "created_at":"2026-09-17T10:00:00Z",
  "expires_at":"2026-09-17T11:00:00Z",
  "record_count":8,
  "page_size":500,
  "complete":true
}
```

Only `complete: true` snapshots are usable. A snapshot is bound to one
consumer, contract version, source cursor, checksum, creation time, expiry
and deterministic record order. The canonical entity-kind order is exactly
`user`, `organisation`, `organisation_membership`, `workspace`,
`workspace_membership`, `product`, `organisation_product_entitlement`,
`user_product_assignment`; records are then ordered by `id`, then relationship
IDs. This order is exposed defensively by the Platform contract package and
must not be changed by a consumer.
The checksum is SHA-256 over the canonical UTF-8 JSON sequence of all records
in that order, with no transport paging metadata. A page cursor is opaque,
snapshot-bound and cannot be used against another snapshot or consumer.

The total snapshot is bounded to 10,000 records, each page to at most 500
records, request bodies to 64 KiB and a response to 4 MiB. Limits are enforced
before materialization or response buffering. A finalized snapshot never
changes; expiry makes it unavailable and requires a new bootstrap.

Records are read with:

```text
GET /api/v1/read-authority/snapshots/{snapshotID}/records?after=<opaque>&limit=<1..500>&entity_kind=<allow-listed>
```

The response repeats the snapshot identity/checksum, returns `records`, an
opaque `next_cursor` (or `null` at completion), and `complete`. A page cannot
claim current authority until the consumer has received every page, verified
the snapshot binding/checksum and separately verified source state.

The records response has this exact envelope; each record uses only the
allow-listed fields for its entity kind:

```json
{
  "version":"platform.read-authority.v1",
  "snapshot_id":"snap_*",
  "source_cursor":"cur_*",
  "checksum":"<lowercase hex>",
  "records":[{"entity_kind":"user","id":"usr_example","status":"active","source_event_id":"evt_example","source_sequence":1,"source_schema_version":1}],
  "next_cursor":null,
  "complete":true
}
```

## Incremental changes and cursors

After bootstrap, the consumer reads committed changes:

```text
GET /api/v1/read-authority/changes?after=<opaque global cursor>&limit=<1..500>
```

The global cursor is opaque to consumers and monotonic at the committed
Platform outbox boundary. It is distinct from each event's per-aggregate
`sequence`. A change response contains the read-authority version, source
state, `changes`, `next_cursor` and `has_more`; each change contains its global
cursor and the existing versioned Platform event envelope. Its exact envelope
is:

```json
{
  "version":"platform.read-authority.v1",
  "state":"current",
  "source_cursor":"cur_2",
  "changes":[{"cursor":"cur_2","event":{"event_id":"evt_example","event_type":"platform.user.created","aggregate_type":"User","aggregate_id":"usr_example","sequence":1,"schema_version":1,"occurred_at":"2026-09-17T10:00:00Z","payload":{"user_id":"usr_example","active":true}}}],
  "next_cursor":"cur_2",
  "has_more":false
}
```

The Platform event payload allow-list remains the authority in
[`platform-events-v1.md`](platform-events-v1.md); this contract does not add
credentials, product roles or billing facts to event payloads.

Consumers must require both properties:

1. global cursors advance without omission or silent skip; and
2. each aggregate's sequence is the next expected sequence, or the consumer
   remains `gap` and requests recovery.

Duplicate event IDs are idempotent only when the complete envelope is
identical. A conflicting duplicate, a missing cursor, an expired cursor or a
retention hole returns `resync_required`; the consumer must not infer a
continuation. Changes are bounded to 500 per response and only committed
outbox rows are visible.

## Freshness and fail-closed states

The status endpoint returns the version, consumer binding, source cursor,
source state, retention horizon and readiness without returning records. Its
exact envelope is:

```json
{
  "version":"platform.read-authority.v1",
  "consumer":"tockrctrl",
  "contract_version":"platform.read-authority.v1",
  "state":"current",
  "ready":true,
  "source_cursor":"cur_2",
  "retention_horizon":"cur_0"
}
```

These states have one meaning:

| State | Meaning | Authoritative reads |
| --- | --- | --- |
| `current` | Snapshot/feed and consumer checkpoint are complete and ordered | Allowed for the covered shared facts |
| `stale` | The consumer has not reached the latest committed source within its freshness bound | Denied for access broadening; retain only for diagnosis/recovery |
| `gap` | An aggregate or global cursor sequence is missing | Denied; request ordered recovery |
| `blocked` | Unknown schema, conflicting identity/envelope or integrity failure | Denied until repaired and revalidated |
| `unavailable` | Source, key configuration or required read surface cannot safely answer | Denied; no stale fallback |
| `resync_required` | Cursor/snapshot retention or continuity cannot prove the next state | Denied; bootstrap a new snapshot |

Only `current` is authoritative. No endpoint may return `current` for an
expired, incomplete, corrupt, conflicting, unavailable, stale or gapped
source. Consumers must fail closed for access decisions when any required
record or relationship is absent, non-current or invalid.

## Errors and limits

Errors have this non-sensitive shape:

```json
{"version":"platform.read-authority.v1","code":"resync_required","state":"resync_required","retryable":false,"resync_required":true,"message":"read authority requires resynchronisation"}
```

`message` is stable operational text. It never contains SQL, key material,
record values, signatures, nonce values, filesystem paths or stack traces.

| Code | HTTP | Retry/resync meaning |
| --- | ---: | --- |
| `version_mismatch` | 406 | Deploy a supported contract; do not downgrade implicitly |
| `unauthenticated` | 401 | Fix machine authentication; browser credentials do not help |
| `forbidden` | 403 | Consumer/key/product binding is not permitted |
| `invalid_request` | 400 | Correct the bounded request |
| `limit_exceeded` | 400 | Reduce page/body/resource request |
| `stale`, `gap`, `blocked`, `unavailable` | 409 | Deny authority; resolve source state |
| `resync_required`, `cursor_expired` | 409 | Bootstrap a fresh snapshot |
| `snapshot_expired`, `snapshot_incomplete`, `checksum_mismatch` | 409 | Discard snapshot and bootstrap again |
| `conflict` | 409 | Preserve the explicit conflict and stop applying |

## Compatibility and rollback

Version 1 has no implicit compatibility with `platform.v1` assertions. Both
CTRL and IMS use the same route/schema/state machine; only the authenticated
consumer key, product key and local projection implementation differ. A
consumer may deploy a new implementation while retaining v1 support, but
Platform must advertise and verify an exact version. Version negotiation never
selects an older version silently.

The minimum source-cursor retention guarantee is seven days, subject to the
published status horizon. A consumer that is offline beyond that horizon must
bootstrap. Snapshots expire no later than 24 hours. A future v2 requires a new
contract artifact, compatibility matrix, migration/rollback plan and
independent acceptance; it cannot reinterpret v1 fields.

To roll back the new runtime, Platform disables the read-authority routes or
reports `unavailable`; it does not delete canonical records, outbox rows or
consumer data, and it never reports stale data as current. Consumers retain
their local projection for diagnosis but deny any decision requiring a
non-current shared fact until a valid snapshot/feed is established.
