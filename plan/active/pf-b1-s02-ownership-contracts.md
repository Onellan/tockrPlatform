# PF-B1-S02 — Platform ownership and shared contracts

Status: **Ready**

## Objective

Freeze Platform ownership, canonical hierarchy, roles, IDs, product-access
semantics and the boundary with CTRL/IMS before runtime implementation.

## Authority and current evidence

Authority is the PF brief and `docs/contracts/platform-contract-v1.md`.
Current evidence is CTRL PA structural/product-access material and IMS PA/PD
material at the exact SHAs recorded in the alignment matrix.

## Affected files/packages

`docs/architecture/platform-ownership-and-boundaries.md`,
`docs/contracts/platform-contract-v1.md`, domain glossary references and the
PF index. Future seams are `internal/domain`, `internal/store` and
`internal/platform/assertion`; no runtime code is in this Slice.

## Ordered work

1. Map User, Organisation, membership, Workspace, Product, entitlement and
### WP01 - Ordered work package
   assignment ownership and prohibited product responsibilities.
Route: kind=other; risk=H[AUTH,GOV,DOC]
2. Freeze role scopes, stable opaque ID rules and the deny-by-default effective
### WP02 - Ordered work package
   access predicate.
Route: kind=authorization; risk=H[AUTH,GOV]
3. Define contract versioning, compatibility and unresolved semantic decision
### WP03 - Ordered work package
   records for later implementation.
Route: kind=other; risk=H[API,DEP,DOC]

## Migration impact

No data changes. Future mapping must be additive and explicit; current CTRL/IMS
IDs are not treated as canonical Platform IDs.

## Security impact

Membership cannot grant product access; entitlement cannot auto-assign users;
product roles and billing data cannot cross the boundary. Unknown contract
versions fail closed at security-sensitive edges.

## Acceptance criteria

- canonical hierarchy and ownership table are unambiguous;
- roles exactly match SYSTEM/system_admin, Organisation owner/admin/member and
  Workspace admin/member/viewer;
- product keys are `product.tockrctrl` and `product.tockrims`;
- access predicate has separate entitlement and assignment decisions;
- CTRL/IMS Projects and product roles are explicitly outside Platform;
- all current source conflicts are documented rather than silently merged.

## Tests and evidence

Run architecture/security/quality profiles; inspect contract terms for both
product keys, all role values, ID prefixes and prohibited billing/product-role
fields; perform independent architecture review of the boundary document.

## Dependencies

PF-B1-S01.

## Stop/go conditions

Stop for any disagreement about shared identity, role meaning or access order
that cannot be resolved by the brief and current source evidence.

## Rollback

Revert only the contract/document commit. No external system or product data is
changed.
