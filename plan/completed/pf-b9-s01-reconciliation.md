# PF-B9-S01 — CTRL/IMS reconciliation inventory and mapping

Status: **Implemented / terminal**

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

## Terminal evidence

Accepted implementation candidate: `eafb9451572d248275f6eafe6174a547a4eceadb`.

Terminal closeout candidate: `9bd9f527d1a1701273f5196fc39e7a008ef50e23`.

- Independent engineering review: **PASS** after sequential repairs for
  cross-source collision propagation and same-source relationship provenance.
- Independent tester acceptance: **PASS** for deterministic proposals,
  collision/ambiguity blocking, exact source/version provenance, redacted
  match-key output, unresolved relationship blocking and read-only CLI
  operation.
- Exact-candidate local validation: `full/local`, format, architecture,
  security, quality, frontend, unit, focused package and race tests, `go vet`
  and diff hygiene **PASS**. Container build profiles are
  **NOT_APPLICABLE** because no authorised Dockerfile exists.
- Platform keeps its initial one-connection SQLite policy. No CTRL/IMS code,
  database, source connector, production record, product role or authority
  cutover was changed.

Detailed evidence:
[`reconciliation`](../../docs/implementation/audits/pf-b9-s01-reconciliation.md),
[`independent review`](../../docs/implementation/audits/pf-b9-s01-independent-review.md),
[`independent acceptance`](../../docs/implementation/audits/pf-b9-s01-independent-acceptance.md).
