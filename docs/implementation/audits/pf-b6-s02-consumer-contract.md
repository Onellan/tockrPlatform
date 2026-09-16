# PF-B6-S02 reconciliation evidence

Accepted implementation candidate: `e9de6b100eafd19ad75a4b4c3046e0107cb93f62`

Terminal closeout candidate: `681a4c82170e3ae0b9abb558a138875235177489`.

Plan: [`plan/completed/pf-b6-s02-consumer-contract.md`](../../../plan/completed/pf-b6-s02-consumer-contract.md)

| Acceptance surface | Evidence | Result |
| --- | --- | --- |
| Audience/product/version contract | Compatibility matrix for `product.tockrctrl`/`tockrctrl` and `product.tockrims`/`tockrims`, version 1 | PASS |
| Consumer verification | Public-key-only Ed25519 verification with issuer, audience, version, scope, time and assertion-ID checks | PASS |
| Failure taxonomy | Unauthenticated, forbidden, stale, unavailable and version-mismatch classes | PASS |
| Negative fixtures | Valid, expired, revoked/retired-key, wrong-audience, wrong-product, stale-scope and version-mismatch cases | PASS |
| Claim and authority boundary | No product roles, billing, passwords, sessions, full entitlements, CTRL/IMS code or cutover | PASS |
| Independent engineering review | Read-only review of exact candidate `e9de6b1` | PASS; no R1 findings |
| Independent tester acceptance | Separate uncached acceptance run on exact candidate `e9de6b1` | PASS |
| Exact-candidate local validation | `full/local` on `681a4c82170e3ae0b9abb558a138875235177489`, including format, architecture, security, migration, frontend, quality, unit, integration, race and container applicability | PASS / NOT_APPLICABLE where declared |
| Container profiles | AMD64 and ARM64 container builds | NOT_APPLICABLE; no authorised Dockerfile exists |

The initial one-connection SQLite policy remains unchanged. No CTRL/IMS route,
code, data, product role, migration or authority cutover was changed. PF-B7 and
all later Batches remain unimplemented.
