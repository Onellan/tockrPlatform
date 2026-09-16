# Tockr Platform events v1

This is the versioned shared Platform event envelope for identity, tenancy and
access changes. It supports durable at-least-once delivery; it does not create
CTRL/IMS product-domain authority or require a broker.

## Envelope

Every committed outbox row contains:

```text
event_id        evt_* opaque unique identity
event_type      platform.* versioned event name
aggregate_type  User | Organisation | Workspace | Product | Access
aggregate_id    canonical Platform ID
sequence        positive monotonic sequence per aggregate
schema_version  1
occurred_at     UTC mutation time
payload         strict JSON object from the event allow-list
published_at    nullable delivery marker
```

`event_id` is the consumer idempotency key. `aggregate_type`, `aggregate_id`
and `sequence` identify ordering; a consumer must reject duplicates and must
not silently apply a gap. The outbox row and owning authority mutation commit
in one SQLite transaction. A failed transaction produces neither the authority
fact nor its outbox row.

## Event allow-list

| Event type | Aggregate | Payload fields |
| --- | --- | --- |
| `platform.user.created` | User | `user_id`, `active` |
| `platform.user.status_changed` | User | `user_id`, `active` |
| `platform.organisation.created` | Organisation | `organisation_id` |
| `platform.organisation.membership_added` | Organisation | `organisation_id`, `membership_id`, `user_id`, `role`, `active` |
| `platform.organisation.membership_role_changed` | Organisation | `organisation_id`, `membership_id`, `user_id`, `role`, `active` |
| `platform.organisation.membership_deactivated` | Organisation | `organisation_id`, `membership_id`, `user_id`, `role`, `active` |
| `platform.organisation.archived` | Organisation | `organisation_id`, `active` |
| `platform.organisation.renamed` | Organisation | `organisation_id`, `name` |
| `platform.workspace.created` | Workspace | `workspace_id`, `organisation_id`, `active` |
| `platform.workspace.membership_added` | Workspace | `workspace_id`, `membership_id`, `user_id`, `role`, `active` |
| `platform.workspace.membership_role_changed` | Workspace | `workspace_id`, `membership_id`, `user_id`, `role`, `active` |
| `platform.workspace.membership_deactivated` | Workspace | `workspace_id`, `membership_id`, `user_id`, `role`, `active` |
| `platform.workspace.archived` | Workspace | `workspace_id`, `active` |
| `platform.product.retired` | Product | `product_key`, `status` |
| `platform.access.entitlement_granted` | Access | `entitlement_id`, `organisation_id`, `product_key`, `status` |
| `platform.access.entitlement_revoked` | Access | `entitlement_id`, `organisation_id`, `product_key`, `status` |
| `platform.access.assignment_granted` | Access | `assignment_id`, `organisation_id`, `user_id`, `product_key`, `status` |
| `platform.access.assignment_revoked` | Access | `assignment_id`, `organisation_id`, `user_id`, `product_key`, `status` |

Payloads contain only canonical opaque IDs, Platform roles/statuses and bounded
shared names. They never contain credentials, password hashes, raw session
tokens, product roles, billing/payment facts, mutation reasons or full
entitlement detail.

## Delivery boundary

`ListPendingPlatformEvents` reads committed rows in bounded batches and
`MarkPlatformEventsPublished` records a delivery marker only for exact pending
event IDs. Delivery remains at-least-once; consumers own idempotency and local
projection state. No synchronous per-request Platform call, shared database,
Kafka/Redis dependency or CTRL/IMS authority cutover is introduced.
