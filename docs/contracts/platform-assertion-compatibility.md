# Platform assertion consumer compatibility v1

This matrix is the bounded consumer contract for PF-B6. CTRL and IMS may
implement their own adapters against it. It does not change either product's
authentication, roles, project authority or data store, and it does not claim
cutover.

## Audience and product matrix

| Platform product key | Assertion audience | Supported assertion version | Consumer authority after validation |
| --- | --- | --- | --- |
| `product.tockrctrl` | `tockrctrl` | `1` | TockrCTRL applies its own product role and Project rules |
| `product.tockrims` | `tockrims` | `1` | TockrIMS applies its own product role and Project/governance rules |

The product key is an adapter configuration value, not an assertion claim. A
consumer must require the expected audience and the exact corresponding
product key from this table; a mismatch is forbidden. No product role,
billing/payment fact, password, session token or full entitlement detail is
accepted from Platform.

## Validation and negotiation

Consumers must:

1. obtain the Platform public verification-key set from
   `/.well-known/tockr-platform-assertion-keys` through their trusted transport
   and retain overlapping keys during rotation;
2. parse the three-segment v1 form strictly and verify Ed25519, key ID, issuer,
   expected audience, assertion version, `usr_`/`org_`/`wsp_` scope, issued/expiry
   time and unique assertion ID;
3. reject unknown versions rather than falling back to an older parser;
4. treat the assertion as a short-lived handoff proof, not as a product session
   or product authorization decision; and
5. apply local product roles and Project/governance rules only after Platform
   validation succeeds.

There is no synchronous per-request Platform dependency, shared database,
identity guess or authentication cutover in this contract. Version negotiation
is explicit: v1 is the first supported version, and unsupported versions fail
closed with `version_mismatch`.

## Failure taxonomy

Consumers should expose only the class, not assertion contents or scope, to
callers and logs:

| Class | Examples | Consumer action |
| --- | --- | --- |
| `unauthenticated` | malformed token, bad signature, unknown/retired key | do not establish product identity |
| `forbidden` | wrong audience/product, invalid or stale scope, signed claim not permitted | deny the product request |
| `stale` | expired, not-yet-valid or replayed assertion ID | require a fresh handoff |
| `unavailable` | verifier/key configuration unavailable | fail closed without claiming product denial |
| `version_mismatch` | unsupported assertion version | fail closed and use an explicitly supported contract version |

The Platform test harness covers valid, expired, retired-key, wrong-audience,
wrong-product, stale-scope and version-mismatch fixtures. A valid assertion
does not bypass consumer authorization.
