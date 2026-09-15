# Tockr Platform coding standards

Product and contract authority wins over implementation convenience. These
standards inherit the strongest common CTRL/IMS rules and are adapted to the
Platform boundary.

## Architecture and locality

- Go modular monolith with `net/http` + Chi and SQLite/WAL.
- `internal/domain` has no HTTP, templ or SQLite dependency.
- `internal/store` exposes narrow caller contracts; one SQLite adapter owns
  representation and transaction details.
- Capability-local files are preferred over generic service/manager/repository
  layers.
- Do not introduce microservices, ORM, Kafka, Redis, service mesh or shared
  CTRL/IMS databases without new authority and evidence.
- Provider, email and network side effects remain outside database transactions.

## Authorization and security

Every protected read and write proves active User, Organisation membership,
entitlement, assignment and Workspace access as applicable. UI hiding is never
authorization. CSRF, secure sessions, bounded HTTP resources, safe error
classification, audit and secret-free logs are mandatory at their relevant
boundaries.

## SQLite and history

Use parameterised SQL, short transactions, WAL and the ordered migration ledger
in `docs/architecture/versioned-database-migrations.md`. Validate material
current state inside the mutation transaction. Preserve historical facts;
never fabricate actor, time, reason or identity matches.

## Presentation

Use real templ layouts/pages/components, source-owned shadcn-templ primitives,
Tailwind and server-rendered HTML. JavaScript is progressive enhancement only.
Presentation components render already-authorised state and do not decide
access or authority.

## Testing and review

Use behaviour-first tests at stable public seams. New behavior and regressions
use RED → GREEN; covered refactors use GREEN-preserving evidence. Migration
changes require fresh, upgrade and reopen tests. Authorization requires allowed
and denied cases. Independent engineering review and independent tester
acceptance are separate gates. Batch and final certification receive full
independent review; every small Slice receives narrow review.

## Validation and publication

`scripts/validate.py` and its registry are the local command authority. Context
failures (`INVOCATION_FAIL`, `ENV_FAIL`, `TOOL_FAIL`, `FIXTURE_FAIL`,
`PREREQUISITE_FAIL`) are not product failures. GitHub Actions is not a required
implementation-validation path. Publication and exact-candidate certification
remain explicit delivery gates.
