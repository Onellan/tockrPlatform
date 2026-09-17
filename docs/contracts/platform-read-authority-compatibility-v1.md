# Platform read-authority v1 compatibility

This companion is the consumer implementation contract for
`platform.read-authority.v1`. It applies equally to TockrCTRL and TockrIMS and
does not authorize either repository to change Platform ownership, import
production data or cut over product authorization.

## Consumer matrix

| Consumer | Machine identity | Product key | Local responsibility |
| --- | --- | --- | --- |
| TockrCTRL | `tockrctrl` | `product.tockrctrl` | Local projection, CTRL product roles/configuration and product requests |
| TockrIMS | `tockrims` | `product.tockrims` | Local projection, IMS product roles/configuration and product requests |

The same version, headers, route schemas, record allow-list, state machine,
cursor rules and negative behavior apply to both consumers. The product key is
only the Platform entitlement identity. It is not a product role and does not
permit Platform to answer product-domain authorization.

## Required bootstrap state machine

Each consumer stores the snapshot metadata, records, checksum, source cursor,
contract version and local state in its own database. A consumer must follow
this sequence:

```text
unavailable/initial
  -> request authenticated snapshot
  -> receive complete immutable snapshot
  -> verify consumer, version, expiry, deterministic order and checksum
  -> atomically install local snapshot and source cursor
  -> consume ordered changes
  -> current
```

The consumer must not expose the projection as current between page receipt
and final checksum verification. Installation is local and atomic. A failed
page, checksum, relationship or provenance check leaves the previous state
non-current and records `blocked` or `resync_required` as appropriate.

The snapshot is a source baseline, not a permission to skip event ordering.
The first change request starts after the snapshot's opaque `source_cursor`.

## Change application rules

For every change, the consumer verifies the exact contract version, event
allow-list, event ID, aggregate kind/ID, positive per-aggregate sequence and
global cursor continuity before applying it. It then applies the event and
advances its local checkpoint in one local transaction. Duplicate identical
events are acknowledged idempotently. Conflicting duplicates, missing global
cursors and aggregate sequence gaps set `blocked` or `gap` and stop access
broadening.

`stale`, `gap`, `blocked`, `unavailable` and `resync_required` all deny
authoritative access decisions. There is no stale-while-authoritative mode and
no fallback to a synchronous Platform request. Recovery is either the
documented ordered retry or a new complete snapshot.

## Authentication compatibility

Consumers sign requests with an Ed25519 private key held by their deployment
secret manager. They send the exact v1 version, consumer, key ID, timestamp,
nonce and base64url signature headers. They sign the raw body digest and the
canonical route path as specified in
[`platform-read-authority-v1.md`](platform-read-authority-v1.md).

Browser sessions, user assertions, product assertions and public assertion-key
lookups are not machine authentication. Consumers must not forward browser
cookies or user tokens to the integration routes. Key rotation uses the
documented overlap; a retired key, replayed nonce, wrong consumer binding,
clock-skewed timestamp or malformed signature is an authentication failure.

## Record compatibility

Consumers must implement only the eight allow-listed entity kinds and must
reject unknown kinds or fields. They must preserve opaque IDs and source
provenance without deriving identity matches from names or email addresses.
They must preserve explicit archived/revoked/retired states and must not
delete them to make a projection appear current.

Platform membership roles (`owner`, `admin`, `member`, `viewer`) are shared
tenancy facts. Product roles, Project permissions, billing/payment status,
document content and mutation reasons remain local or outside this contract.

## Rollout, migration and rollback

The first supported version is v1. A consumer rollout must deploy the v1
decoder, strict negative handling and durable state/checkpoint storage before
enabling the feed. Platform and consumer implementations may overlap during a
future version rollout, but an unknown version fails closed and no implicit
downgrade is permitted.

If a consumer cannot prove a complete snapshot, ordered cursor, current source
state or valid authentication, it reports its projection non-current and
denies decisions needing the affected shared fact. It may continue serving
non-authoritative diagnostics and product data that does not depend on that
fact, subject to its own product policy. It must never make a synchronous
Platform read a hidden recovery path.

If Platform disables the integration surface, it reports `unavailable` or a
non-success response. Existing canonical Platform data and local consumer
history remain intact. Re-enable requires the normal authentication,
snapshot/feed continuity and acceptance checks; rollback does not erase
history or change the v1 assertion contract.

## Required negative cases

Both consumer compatibility suites must prove rejection of:

- `platform.v1` or unknown read-authority versions;
- an unknown entity kind or forbidden field;
- a duplicate or out-of-order record/change;
- a missing global cursor or aggregate sequence gap;
- stale, blocked, unavailable, expired or incomplete snapshots;
- checksum or source-cursor mismatch;
- browser/user assertion in place of machine signing;
- unknown, retired, cross-consumer or malformed signing keys;
- timestamp skew and nonce replay; and
- a response that contains secrets, product roles, billing facts or document
  bytes.

The test suites must bind their results to the exact Platform contract version
and implementation candidate. A missing test tool, fixture or environment is
`BLOCKED / NOT RUN`, never acceptance.
