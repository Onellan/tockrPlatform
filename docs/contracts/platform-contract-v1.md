# Tockr Platform contract v1

This document is the versioned cross-repository contract planned by PF. It is
not a runtime implementation and does not authorise CTRL/IMS cutover.

## Canonical records

```text
User: usr_* opaque ID, active lifecycle, authentication authority in Platform
Organisation: org_* opaque ID, active lifecycle
Workspace: wsp_* opaque ID, owned by exactly one Organisation
OrganisationMembership: User + Organisation + Platform role + lifecycle
WorkspaceMembership: User + Workspace + Platform role + lifecycle
Product: stable key and lifecycle
OrganisationProductEntitlement: Organisation + Product + lifecycle
UserProductAssignment: User + Organisation + Product + lifecycle
```

Initial product keys are `product.tockrctrl` and `product.tockrims`.

## Effective access

```text
active User
∧ active OrganisationMembership
∧ active OrganisationProductEntitlement
∧ active UserProductAssignment
∧ active Workspace ownership/access
```

The product then applies its own product-role and Project authorization. Platform
does not inspect billing/payment details or product roles to make the shared
decision.

## Assertion envelope

The planned short-lived handoff assertion contains only:

```text
issuer
audience
platform_user_id
platform_organisation_id
platform_workspace_id
issued_at
expires_at
assertion_id
assertion_version
```

It excludes product roles, billing, passwords, session tokens and full
entitlement detail. The consumer must validate signature, issuer, audience,
version, expiry and the active access predicates at the Platform boundary.

The PF-B6-S01 wire form is three URL-safe base64 segments: a strict JSON
header, a strict JSON payload and an Ed25519 signature over `header.payload`.
The header contains `alg=Ed25519`, the configured key ID and
`typ=TockrPlatformAssertion`. Version 1 uses the exact allow-listed payload
above, a two-minute default lifetime bounded by a fifteen-minute deployment
maximum, configured consumer audiences (`tockrctrl` and/or `tockrims`), and
opaque assertion IDs with `ast_` prefix. Consumers must reject unknown key IDs,
algorithms, versions, audiences, expired/future assertions and replayed
assertion IDs. Public verification keys are exposed through
`/.well-known/tockr-platform-assertion-keys`; the endpoint contains public
material only and supports overlap during key rotation.

## Events and projections

Platform events use a versioned envelope with event ID, aggregate type/ID,
sequence, occurred-at, schema version and redacted payload. Consumers persist
inbox identity and last applied sequence; duplicate and out-of-order delivery
is handled explicitly. Events are integration support, not an excuse for a
shared database or synchronous Platform call on every product request.

## Compatibility

Every contract change must state compatibility, consumer impact, migration
strategy, rollback and the first supported version. Unknown versions fail
closed at security-sensitive boundaries. The PF-B6 consumer audience/product
matrix and failure taxonomy are maintained in
[`platform-assertion-compatibility.md`](platform-assertion-compatibility.md).
