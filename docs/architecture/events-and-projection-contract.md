# Platform events and projection support

Platform owns the event contract needed to distribute shared identity and
access changes. It does not own product-domain events.

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

The event is emitted only after the owning transaction commits, and the outbox
record is part of that transaction. Delivery is at-least-once; consumers must
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
