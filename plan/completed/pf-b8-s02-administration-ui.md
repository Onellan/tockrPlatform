# PF-B8-S02 — Platform administration UI

Status: **Implemented / terminal**

## Objective

Deliver server-rendered Organisation Admin, Workspace Admin and System Admin
surfaces with exact Platform authority boundaries.

## Authority and current evidence

Authority is the initial UI tree and role contract. Product administration and
operational screens in CTRL/IMS remain outside Platform.

## Affected files/packages

Admin pages/layouts/components, HTTP authorization routes, view models, CSRF,
audit reads and browser/accessibility tests.

## Ordered work

1. Map General/Members/Workspaces/Products surfaces to narrow commands and
### WP01 - Ordered work package
   authorized read models.
Route: kind=ui; risk=H[UI,AUTH,API]
2. Implement owner/admin/member, Workspace admin/member/viewer and system-admin
### WP02 - Ordered work package
   action visibility plus server-side denial.
Route: kind=ui; risk=H[AUTH,GOV,UI]
3. Validate responsive/accessibility/error/success/history states and no product
### WP03 - Ordered work package
   role/billing leakage.
Route: kind=ui; risk=H[UI,AUTH,HIST,FIN]

## Migration impact

No data import. UI may expose only Platform records and explicit audit facts.

## Security impact

High: IDOR, CSRF, privilege escalation and cross-tenant disclosure are tested
through direct requests, not only browser controls.

## Acceptance criteria

Each initial admin screen exists at the correct scope, actions are authorized,
history is visible without fabrication, and CTRL/IMS operational pages are not
present.

## Tests and evidence

HTTP authorization/CSRF matrix, frontend/profile checks, independent browser
journeys and Batch B8 architecture review.

## Dependencies

PF-B8-S01, PF-B5-S02 and PF-B7-S02.

## Stop/go conditions

Stop if an admin action needs product role authority, billing/payment data or a
client-only state decision.

## Rollback

Disable the affected admin route and preserve already committed authority/audit
facts; do not roll back by deleting membership history.

## Terminal evidence

Accepted implementation candidate: `13315d0fbb2c3b2163f9b34c4f8449de4cefb735`.

- Independent engineering review: **PASS**; no R1 or R2 finding remains.
- Independent tester acceptance: **PASS** for Organisation General/Members/
  Workspaces/Products, Workspace administration, System Admin catalogue,
  server-side authorization, CSRF, history, responsive presentation and
  Platform boundary.
- Exact-candidate local validation: format, frontend, architecture, security,
  quality, unit and integration children **PASS**. The repository composite
  `full/local` race child exceeded its fixed 300-second bound and is recorded
  as **TIMEOUT**, not PASS; the extended changed-package race command passed.
- Browser evidence: signed-in wide and narrow administration journeys passed
  semantic snapshot, visual and console inspection with zero errors/warnings.
- Container build profiles: **NOT_APPLICABLE** because no authorised
  Dockerfile exists.
- The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  production code, data, migration, product role or authority cutover was
  changed.

Detailed evidence:
[`reconciliation`](../../docs/implementation/audits/pf-b8-s02-administration.md),
[`independent review`](../../docs/implementation/audits/pf-b8-s02-independent-review.md),
[`independent acceptance`](../../docs/implementation/audits/pf-b8-s02-independent-acceptance.md).
