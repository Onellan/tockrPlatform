# Platform membership command v1

`platform.membership-command.v1` is the bounded machine command contract for
CTRL and IMS. It permits only `OrganisationMembership` and generic
`WorkspaceMembership` add, role-change and deactivate mutations.

The request is `POST /api/v1/membership-commands` and uses the same signed
request shape as the read-authority transport, with the command version header
and the consumer, key, timestamp, nonce, signature and actor-assertion headers.
The signature canonical string appends the SHA-256 digest of the actor
assertion after the body digest, binding the proof to the product request. The
body is:

```json
{
  "scope": "organisation_membership",
  "operation": "add",
  "organisation_id": "org_example",
  "user_id": "usr_example",
  "role": "member",
  "reason": "approved access change",
  "idempotency_key": "ctrl-2026-09-22-1",
  "expected_version": 0
}
```

The `X-Tockr-Platform-Actor-Assertion` header carries a Platform-issued v1
assertion. Platform verifies the configured consumer signature and actor proof
independently, binds the assertion audience and scope to the request, then
rechecks the actor's live Platform membership before mutation. Caller-supplied
user IDs or product roles are never actor proof.

Role-change and deactivate requests use the current membership version as
`expected_version`; add uses zero. A stable `idempotency_key` with the same
request returns the original result. Reuse with a different request or a stale
version returns `409 conflict` without changing state. Membership, audit and
the existing allow-listed outbox event commit atomically. Reasons remain in
private audit details and are never added to the consumer feed.

The browser session and CSRF administration routes remain unchanged. This API
does not authorize Organisation or Workspace lifecycle, product entitlement or
assignment, product-role, migration, authentication or production cutover
operations.
