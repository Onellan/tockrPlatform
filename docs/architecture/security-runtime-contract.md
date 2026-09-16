# Platform security and runtime contract

The runtime target must match or exceed the current CTRL/IMS security baseline:

- secure, revocable sessions with hashed opaque tokens;
- CSRF protection for state-changing browser requests;
- secure cookie options and restrictive security headers;
- server-side authorization at every protected read/write boundary;
- rate limiting and failed-login backoff;
- MFA/recovery where the applicable authentication flow requires it;
- parameterised SQL and bounded transactions;
- audit records for security and authority changes;
- bounded HTTP header/body/read/write/idle resources;
- safe external error classification;
- no credentials or secrets in logs.

The deployment target is Linux AMD64 and ARM64, with a non-root container,
dropped capabilities, read-only root filesystem, bounded writable `/tmp`,
persistent data volume, graceful shutdown and explicit `/healthz` and `/readyz`.

PF-B10-S01 implements these runtime conditions in the Platform checkout. Final
programme certification remains PF-B10-S02; this contract does not authorize
CTRL/IMS migration, production-data import or authority cutover.
