# Tockr Platform local projection inbox v1

This is the bounded local projection support for Platform event consumers. It
does not make a projection authoritative and it does not grant product access.
Products remain owners of their local projection data and product authorization.

## Durable records

The Platform SQLite store keeps two records for each consumer/aggregate lane:

- the inbox is keyed by `consumer_key + event_id` and retains the complete
  bounded event envelope, schema version, receipt time, state and a redacted
  reconciliation reason;
- the checkpoint records the last applied aggregate sequence and explicit
  `current`, `stale`, `gap`, `blocked` or `unavailable` state.

Unknown source schema versions and invalid current-version envelopes are
retained as `blocked`; they are never discarded to make a projection appear
current. A duplicate event ID is harmless only when its envelope is identical;
an identity conflict fails closed.

## Ordered lifecycle

1. `IngestPlatformProjectionEvent` validates the bounded source envelope and
   records it as pending, stale, gap or blocked.
2. The consumer applies a pending event through its own local projection
   boundary and calls `MarkPlatformProjectionEventApplied` only after that
   local work succeeds.
3. `ReconcilePlatformProjection` promotes only the next durable gap event and
   does bounded work. Missing sequences remain `gap`.
4. `MarkPlatformProjectionUnavailable` records an explicit unavailable state;
   it does not silently fall back to stale or current. An operator may call
   `ResumePlatformProjection` after the source is available again; the state is
   recomputed from durable inbox evidence and never guessed as current.

An event cannot advance a checkpoint out of order. A blocked or unavailable
checkpoint cannot be applied through the Platform inbox API. Reads that depend
on a projection must treat every state except `current` as non-authoritative.

## Boundary and retention

Payloads remain the v1 Platform allow-list and contain no credentials, raw
session tokens, product roles, billing facts or mutation reasons. Inbox replay
is bounded to 1..1000 records per call. The migration is ordered, fresh/upgrade/
reopen safe and retains unknown source versions for later repair.

This support adds no broker, shared product database, synchronous product
request dependency, CTRL/IMS code or authority cutover.
