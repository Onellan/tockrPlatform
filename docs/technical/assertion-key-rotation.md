# Platform assertion key rotation

Platform handoff assertions use Ed25519 signing. Private signing material is
startup configuration and must be supplied through the deployment secret
boundary; it is never committed, logged or returned by the Platform API.

Required configuration:

| Variable | Meaning |
| --- | --- |
| `PLATFORM_ASSERTION_ISSUER` | Exact issuer string placed in assertions and checked by consumers |
| `PLATFORM_ASSERTION_KEY_ID` | Active signing key identifier |
| `PLATFORM_ASSERTION_PRIVATE_KEY` | Active Ed25519 private key, encoded as 64-byte hex or base64 |
| `PLATFORM_ASSERTION_AUDIENCES` | Comma-separated, explicitly allowlisted consumer audiences |
| `PLATFORM_ASSERTION_VERIFY_KEYS` | Optional comma-separated `key_id=public_key` entries retained for verification during rotation |

The process refuses to start when issuer, active key, private key or audience
configuration is absent or malformed. Assertions are version 1, contain only
the shared identity/scope claims in `platform-contract-v1`, and expire after
two minutes by default. The configured maximum lifetime is bounded at fifteen
minutes.

## Rotation procedure

1. Generate a new Ed25519 key in the deployment secret boundary and assign a
   new, never-reused key ID.
2. Deploy the new private key as `PLATFORM_ASSERTION_PRIVATE_KEY` and retain
   the previous public key in `PLATFORM_ASSERTION_VERIFY_KEYS`.
3. Confirm the public-key endpoint
   `/.well-known/tockr-platform-assertion-keys` exposes the new and retained
   verification keys, without exposing private material.
4. Deploy or refresh consumers so they trust the new key before the previous
   key is retired. Keep the old public key available for at least the maximum
   assertion lifetime plus the consumer rollout allowance.
5. Remove the old public key only after that window and record the change in
   the deployment audit. Unknown key IDs and algorithms fail closed.

Key rotation does not upgrade existing sessions or authorise CTRL/IMS cutover.
Consumers must continue to validate issuer, audience, version, signature,
expiry, assertion ID and active shared access according to the versioned
contract.
