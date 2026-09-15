# Tockr Platform Technical Architecture

Status: **Foundation planning authority.** This document describes the
approved target and current foundation state; it is not evidence that Platform
runtime behaviour exists.

## Authority

Product and security meaning in this repository is constrained by:

1. the user-authorised Platform Foundation Programme;
2. `docs/architecture/platform-ownership-and-boundaries.md`;
3. `docs/contracts/platform-contract-v1.md`;
4. `docs/technical/coding-standards.md` and the local validation contract;
5. the active Slice plan.

CTRL and IMS remain authoritative for their own product domains. The current
main SHAs inspected for this foundation are recorded in
`docs/architecture/source-alignment-matrix.md`.

## Target runtime shape

```text
Browser
  ↓
Go + net/http + Chi boundary
  ↓
capability-local Platform application modules
  ↓
narrow caller store contracts
  ↓
one concrete SQLite adapter (platform.db, WAL)
```

The target is a Go modular monolith. It is one deployable Platform process with
SQLite local persistence, not a collection of services.

## Target package responsibilities

```text
cmd/platform                 startup, configuration and graceful shutdown
internal/domain              canonical Platform concepts and invariants
internal/store               narrow capability-local caller contracts
internal/db/sqlite           schema, ordered migrations, queries, transactions
internal/auth                credentials, sessions, CSRF and authentication policy
internal/platform/http       routing, request boundaries and server-rendered UI orchestration
internal/platform/config     strict configuration and production validation
internal/platform/email      bounded email provider seam
internal/platform/assertion  signed product identity assertions
internal/events               event envelopes, outbox and projection contracts
web/layouts                   templ layouts
web/pages                     templ pages
web/components                Tockr semantic components
web/primitives                source-owned shadcn-templ primitives
web/static                    generated Tailwind CSS and bounded progressive enhancement
```

Only a package with a real responsibility is created. There is no generic
service/manager/repository layer and no shared database with CTRL or IMS.

## Non-negotiable boundaries

- Platform owns User, authentication, Organisation, memberships, Workspace,
  Product, entitlement, assignment, shared audit and Platform integration
  contracts.
- CTRL owns CTRL Projects, CTRL domain rules and product-specific roles.
- IMS owns IMS Projects, IMS governance, Framework and product-specific roles.
- Product access requires both Organisation entitlement and User assignment;
  Organisation membership alone is insufficient.
- Authorization, CSRF and business rules are server-side.
- Historical facts are never fabricated to complete a migration.
- No billing/payment authority, product operational screen or product-domain
  Project is implemented by PF.

## Foundation state

Only documentation, agent/skill configuration, validation scaffolding and
implementation plans are in scope for the current foundation change. Application
code, database schema, migrations, runtime containers and migrations of CTRL/IMS
data remain future Slice work.
