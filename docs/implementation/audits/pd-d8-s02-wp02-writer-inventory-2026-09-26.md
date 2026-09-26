# PD-D8-S02 WP-PD8S02-02 — Sole shared-authority writer inventory

**Date:** 2026-09-26  
**Scope:** Platform, TockrCTRL and TockrIMS exact published `main` candidates  
**Decision:** Platform is the sole shared-authority writer whenever the
Platform-authoritative runtime mode is selected. CTRL and IMS retain local
compatibility writers only in the separately selected pre-cutover `local`
mode; the two modes cannot be active together. This evidence does not perform
a production configuration change.

## Exact candidates

| Repository | Published `main` | Evidence source |
| --- | --- | --- |
| Platform | `3115caad66b4086535720a03ccf08c2766666edd` | `git rev-parse HEAD`, clean `main` |
| CTRL | `4710090b2d6019bc1686945828bc1e90eb7f6a8d` | `git rev-parse HEAD`, clean `main` |
| IMS | `69344937bcb8e030976dd2c67447d7d2d41861a8` | `git rev-parse HEAD`, clean `main` |

## Writer inventory

### Platform shared-authority writers

Platform owns the canonical write interfaces and their SQLite implementations:

- `internal/store/identity.go` / `internal/db/sqlite/users.go`: user create
  and active-state changes.
- `internal/store/organisation.go` / `internal/db/sqlite/organisation.go`:
  Organisation create, rename, archive and OrganisationMembership add, role
  change and deactivate.
- `internal/store/workspace.go` / `internal/db/sqlite/workspace.go`:
  Workspace create, archive and WorkspaceMembership add, role change and
  deactivate.
- `internal/store/product.go`, `product_access.go` and
  `internal/db/sqlite/product*.go`: product retirement, Organisation
  entitlement and User product assignment/revocation.
- `internal/store/membership_command.go` and
  `internal/db/sqlite/membership_command.go`: the single machine-authenticated
  command boundary for bounded OrganisationMembership and generic
  WorkspaceMembership mutations.
- `internal/db/sqlite/events.go` and `internal/store/events.go`: the durable
  authoritative outbox that feeds consumer projections.

Read-authority snapshot/checkpoint and projection-state writes are Platform
operational state, not a second consumer authority. They are emitted from the
same Platform source records and are consumed asynchronously by CTRL/IMS.

### CTRL writers and gates

- `internal/platform/http/platform_membership_write.go` is the only
  authoritative shared-membership request path. In `platform` write mode it
  validates projection health, mappings and the bound actor assertion, sends a
  signed Platform command, then applies only the returned state to the local
  projection and audit log.
- `internal/db/sqlite/organization_membership.go`,
  `workspace_membership.go` and `workspace.go` retain local compatibility
  writers. They are selected only when
  `TOCKR_PLATFORM_MEMBERSHIP_WRITE_MODE=local`.
- `internal/db/sqlite/platform_read_authority*.go` and the
  `ApplyPlatform*Projection` methods write only local read projections,
  checkpoints and inbox/audit metadata; they do not write Platform authority.
- CTRL product writers remain local: `projects.go`, `project_access.go`,
  `project_controls.go`, `project_changes.go`, `baselines.go`, progress and
  forecast stores, WBS/timesheet/reporting stores, and
  `invoices.go`/commercial stores.
- `internal/platform/http/server.go` constructs disabled Platform access and
  command seams by default. It enables the command seam only after the
  explicit `platform` write-mode configuration validates.

### IMS writers and gates

- `internal/platform/http/platform_membership_write.go` has the same bounded
  authoritative command path and local projection application as CTRL.
- `internal/db/sqlite/workspace_project_roles.go`,
  `workspace_membership.go`, `workspace_foundation.go` and
  `organisation_administration.go` retain local compatibility writers. They
  are selected only in the explicit `local` write mode.
- `internal/db/sqlite/platform_read_authority*.go` and
  `platform_membership_projection.go` write only the IMS local projection and
  synchronization metadata.
- IMS-owned writers remain local across the product domain: `project_*.go`,
  `project_technical_authority.go` (engineering disciplines),
  `programme_portfolio_author.go`, framework/governance and project control
  stores, `project_variation.go`, `project_contract.go`, and
  `project_notice_claim_dispute.go` (commercial records).
- `internal/platform/http/server.go` constructs disabled access and command
  seams by default and enables commands only for validated explicit platform
  write mode.

## Request-path proof

1. CTRL and IMS normal authenticated requests obtain identity, tenant,
   membership, entitlement and Workspace facts from their local SQLite
   projection/read stores. The projection readers fail closed on stale, gap,
   unavailable or incomplete state.
2. Both production HTTP constructors initialise `platformAccess` and
   `platformCommands` with `platformclient.NewDisabledSeam()`. The only
   request-bound Platform calls are explicit Platform session bootstrap and
   explicit shared-membership admin commands. Product requests do not call
   Platform synchronously.
3. The shadow comparison hook runs from the authenticated boundary but cannot
   perform network I/O in production: its Platform lookup seam remains the
   disabled seam. It compares local decisions only when an explicit test or
   future operator wiring injects a lookup implementation.
4. `platform` and `local` membership-write modes are mutually exclusive
   validated configuration values. Therefore a consumer cannot be an enabled
   shared-authority writer at the same time as Platform. Local compatibility
   methods are rollback/cutover adapters, not a second steady-state writer.
5. Product-owned project, discipline, programme, control and commercial
   mutations remain on consumer-local stores and transactions. Platform feed
   application never touches those tables.

## Evidence commands

The exact candidate focused suites passed:

```text
Platform: go test ./internal/db/sqlite ./internal/platform/http ./internal/platform/membershipcommand ./internal/platform/readauthority ./internal/store -count=1
CTRL:    go test ./internal/platform/config ./internal/platform/http ./internal/platform/client ./internal/platform/sync ./internal/db/sqlite -count=1
IMS:     go test ./internal/platform/http ./internal/db/sqlite ./internal/platform/client ./internal/platform/sync ./internal/platform/config -count=1
```

IMS runtime stylesheet was generated by its checked-in frontend tool before
the Go suite; the generated file is ignored by design and is not product
source.

Repository-wide `go test ./... -count=1 -timeout=25m` also passed on all three
exact candidates (Platform, CTRL and IMS).

## Verdict

**PASS.** The source inventory finds one shared authority (Platform), two
explicitly gated consumer compatibility modes, no enabled dual writer, local
projection reads for normal consumer requests, no ordinary synchronous
Platform dependency, and no Platform writes into CTRL/IMS product-owned
project, discipline, programme, control or commercial data.

