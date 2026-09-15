# PF-B3-S02 — Organisation administration seams

Status: **Implemented / terminal.**

## Objective

Expose narrow, auditable Organisation administration commands and read models
that later UI and product launchers can use without leaking product authority.

## Authority and current evidence

Authority is PF organisation ownership, canonical roles and server-side
authorization. Current CTRL/IMS administration routes are evidence of patterns,
not routes to copy.

## Affected files/packages

Organisation store contracts, HTTP handlers, view models, audit reads and
focused contract tests; no CTRL/IMS route changes.

## Ordered work

1. Define caller contracts for general settings, members and Workspace entry
### WP01 - Ordered work package
   points with least-knowledge surfaces.
Route: kind=authorization; risk=H[AUTH,API]
2. Implement authorization matrices for owner/admin/member and system admin.
### WP02 - Ordered work package

Route: kind=authorization; risk=H[AUTH,GOV]
3. Prove redaction, audit continuity and safe errors for denied/unknown scopes.
### WP03 - Ordered work package

Route: kind=authorization; risk=H[AUTH,HIST,OPS]

## Migration impact

No import. Read models may expose only Platform-owned facts and explicit source
provenance.

## Security impact

Prevent IDOR, membership enumeration and privilege escalation. UI affordances
are not a substitute for HTTP authorization.

## Acceptance criteria

Organisation admin can manage only permitted shared records; members cannot
perform admin mutations; all changes are auditable and product roles remain
absent from the Platform contract.

## Tests and evidence

HTTP authorization matrix, CSRF tests for mutations, redaction/error tests,
audit assertions and independent review.

## Dependencies

PF-B3-S01.

## Stop/go conditions

Stop if an endpoint needs product-specific permission or billing facts, or if a
read model requires a shared product database.

## Rollback

Disable the affected command/read route and retain committed audit/history; no
destructive rollback of membership data.

## Terminal evidence

Accepted implementation candidate: `5181e76de4b3feeb22b9bcc18b3929014915b286`.

- Independent engineering review: **PASS** with no R1 findings after the
  Workspace entry and invalid-limit boundary repair.
- Independent tester acceptance: **PASS** for the HTTP owner/admin/member and
  explicit system-admin matrix, CSRF-protected mutations, safe unknown and
  cross-Organisation errors, redacted Platform-only read models, audit reads,
  audit continuity and product-role absence.
- Exact-candidate local validation: `full/local` **PASS**; format, architecture,
  security, migration, frontend, quality, unit, SQLite/HTTP integration and
  race profiles **PASS**.
- Schema v4 fresh, upgrade, close/reopen and divergent-ledger evidence: **PASS**.
- The system-admin record is explicit and never bootstrapped or inferred by an
  HTTP route. The Workspace entry is an authenticated Platform seam with
  `available=false` until PF-B4 owns Workspace records; no Workspace data was
  fabricated.
- Container build profiles are **NOT_APPLICABLE** because no Dockerfile is in
  this Slice's authorised scope.
- No CTRL/IMS route, code, data, migration, product role or authority cutover
  was changed.
