# PF-B9-S01 — CTRL/IMS reconciliation inventory and mapping

Status: Planned

## Objective

Create read-only reconciliation tooling that inventories CTRL/IMS Users,
Organisations, Workspaces and memberships and proposes deterministic mappings to
Platform opaque IDs.

## Authority and current evidence

Authority is the no-fabrication ID rule and PF migration boundary. Source
schemas and product PA/PD records at the exact SHAs are evidence only; they do
not authorize import.

## Affected files/packages

Future `internal/platform/reconciliation`, source adapters/fixtures, mapping
records, reports, CLI and tests. No production data source is touched by this
Slice.

## Ordered work

1. Define source identity inventory, confidence, collision and ambiguous-review
### WP01 - Ordered work package
   record shapes.
Route: kind=migration; risk=H[DATA,HIST,AUTH,DOC]
2. Implement deterministic dry-run mapping with stable Platform IDs and no
### WP02 - Ordered work package
   guessed actor/time/reason.
Route: kind=migration; risk=H[DATA,HIST,GOV]
3. Prove repeatability, collision visibility, source provenance and redaction.
### WP03 - Ordered work package

Route: kind=migration; risk=H[DATA,AUTH,OPS]

## Migration impact

Read-only by default; no production import, no source mutation and no authority
cutover. Historical facts remain in source systems.

## Security impact

Protect exported identity data, minimize reports and never log credentials or
full personal data unnecessarily.

## Acceptance criteria

Same inputs produce same mapping proposal; ambiguous matches stop for explicit
review; every proposed mapping retains source/version/provenance and no mapping
is silently accepted.

## Tests and evidence

Fixture inventory, collision/ambiguity matrix, repeatability hash, secret scan,
read-only operational proof and independent review.

## Dependencies

PF-B1-S02, PF-B5-S02 and PF-B7-S02.

## Stop/go conditions

Stop on identity ambiguity, unavailable source authority, production access
without explicit authorization or any need to fabricate historical facts.

## Rollback

Delete only disposable dry-run output under its scoped workspace; retain source
systems and Platform records untouched.
