# PF-B8-S02 — Platform administration UI

Status: Planned

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
