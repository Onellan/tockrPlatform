# PF-B5-S01 — Product catalogue and Organisation entitlements

Status: Planned

## Objective

Create the Platform Product catalogue and independent
OrganisationProductEntitlement authority for `product.tockrctrl` and
`product.tockrims`.

## Authority and current evidence

Authority is the product-access contract. CTRL PD and IMS PD independently
identify product keys and separate Organisation entitlement from user
assignment; neither delegates billing authority to a product.

## Affected files/packages

Product and entitlement domain/store/SQLite files, system-admin HTTP commands,
audit and tests.

## Ordered work

### WP01 - Ordered work package
1. Define stable Product key/lifecycle and system-admin ownership.
Route: kind=authorization; risk=H[AUTH,GOV,API]
2. Implement entitlement lifecycle independently of membership and assignment.
### WP02 - Ordered work package

Route: kind=authorization; risk=H[AUTH,DATA,HIST]
3. Prove entitlement changes are audited and do not auto-assign users or expose
### WP03 - Ordered work package
   billing/payment facts.
Route: kind=authorization; risk=H[AUTH,FIN,DOC]

## Migration impact

Fresh/upgrade/reopen ledger tests. Catalogue is Platform-owned and must not
rewrite product history in CTRL/IMS.

## Security impact

Only system/admin-authorized callers manage catalogue/entitlements. Entitlement
is not authentication, membership or product-role authorization.

## Acceptance criteria

Both initial product keys exist as independent catalogue records; an
Organisation can be entitled without assigning every user; entitlement status
is active-state checked and audited.

## Tests and evidence

Role/authorization matrix, entitlement/assignment separation tests, migration
tests, redaction checks and independent review.

## Dependencies

PF-B4-S02.

## Stop/go conditions

Stop if billing data is required, if entitlement auto-assigns membership, or if
product keys are treated as mutable display labels.

## Rollback

Revoke/restore entitlement through forward audited transitions; never delete the
catalogue or invent a historical billing reason.
