# Platform ownership and boundaries

## Canonical shared authority

Tockr Platform owns the following shared concepts:

```text
User
Authentication
Organisation
OrganisationMembership
Workspace
WorkspaceMembership
Product
OrganisationProductEntitlement
UserProductAssignment
shared Platform audit
shared identity assertions and Platform events
```

The canonical hierarchy is:

```text
User
  ↓
OrganisationMembership
  ↓
Organisation
  ↓
Workspace
  ↓
WorkspaceMembership
```

Products are independent consumers:

```text
Tockr Platform
├── shared identity / tenancy / access
├── TockrCTRL: CTRL Projects and CTRL domain
└── TockrIMS: IMS Projects and IMS domain
```

Platform does not own CTRL or IMS Projects, product-specific roles, governance,
Frameworks, project controls, invoices or product billing.

## Roles

Platform roles are fixed as:

| Scope | Roles |
| --- | --- |
| System | `system_admin` |
| Organisation | `owner`, `admin`, `member` |
| Workspace | `admin`, `member`, `viewer` |

`SYSTEM` is a scope marker, not a user-facing role string. Product-specific
roles remain in CTRL/IMS and must not appear in Platform assertions.

## Access rule

An effective product request requires:

```text
authenticated active User
  → active OrganisationMembership
  → active OrganisationProductEntitlement
  → active UserProductAssignment
  → valid Workspace ownership/access
  → product-owned authorization
```

Membership alone never grants product access. Entitlement never auto-assigns
every member. Billing/payment facts are not required by a product to decide
access.

## ID and history rules

Platform IDs are stable opaque IDs with a type prefix (`usr_`, `org_`, `wsp_`).
They are not derived from current CTRL/IMS IDs. Reconciliation records source
identity, confidence, decision actor, decision time and reason only when those
facts are known. Unknown historical facts remain unknown.
