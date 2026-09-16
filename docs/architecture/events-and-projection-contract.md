# Platform events and projection support

Platform owns the event contract needed to distribute shared identity and
access changes. It does not own product-domain events.

The v1 implementation and payload allow-list are defined in
[`platform-events-v1.md`](../contracts/platform-events-v1.md). The envelope is
stored in Platform's durable outbox before delivery.

## Event envelope

```text
event_id        stable opaque event identity
event_type      namespaced shared event name
aggregate_type  User | Organisation | Workspace | Product | Access
aggregate_id    Platform opaque ID
sequence        monotonic per aggregate
schema_version  explicit contract version
occurred_at     committed UTC timestamp
payload         minimum shared, non-secret facts
```

The outbox record is written in the owning transaction and becomes observable
only after that transaction commits. Delivery is at-least-once; consumers must
be idempotent. No event contains credentials, raw session tokens, product roles
or billing details.

## Projection rules

- consumer inbox identity and aggregate sequence are durable;
- duplicate events are harmless;
- gaps are detected and reconciled rather than silently skipped;
- stale or unavailable projections are classified explicitly;
- product authorization remains local to the product after shared access is
  established;
- reconciliation can prove source/version/decision provenance.
