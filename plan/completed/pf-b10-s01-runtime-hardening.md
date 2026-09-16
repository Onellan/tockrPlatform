# PF-B10-S01 — Security and hardened runtime

Status: **Implemented / terminal**

## Objective

Certify Platform security controls, bounded HTTP behavior, health/readiness and
AMD64/ARM64 hardened container runtime.

## Authority and current evidence

Authority is `security-runtime-contract.md`, current CTRL/IMS runtime standards
and the local validation registry. Runtime implementation is now recorded by
the candidate-bound evidence below; PF-B10-S02 remains the separate final
certification boundary.

## Affected files/packages

HTTP/server lifecycle, config, security middleware, Dockerfile/Compose/runtime
docs, health endpoints, startup/migration and security tests.

## Ordered work

1. Implement strict production config, headers, limits, graceful shutdown,
### WP01 - Ordered work package
   health/readiness and safe errors.
Route: kind=other; risk=H[AUTH,OPS,API]
2. Harden container/user/capabilities/rootfs/tmp/volume and build targets.
### WP02 - Ordered work package
Route: kind=other; risk=H[DEPLOY,OPS,AUTH]
3. Run security, migration, race, AMD64 and ARM64 evidence with failure
### WP03 - Ordered work package
   classification.
Route: kind=other; risk=H[AUTH,DATA,CONC,DEPLOY]

## Migration impact

Startup preserves exact migration prefixes and refuses divergence; runtime
readiness checks the existing Platform database handle. No migration version or
database schema changed in this Slice.

## Security impact

High and cross-cutting: no secrets in image/logs, no root, bounded resources,
revocation/CSRF/rate limits and fail-closed authorization remain mandatory.

## Acceptance criteria

All runtime contract controls are directly evidenced on both architectures;
`/healthz` and `/readyz` are distinct, safe and operational; container is
non-root/capability-dropped/read-only-root with persistent data.

## Tests and evidence

All material local profiles, container smoke, migration fresh/upgrade/reopen,
security negative matrix, race, runtime resource checks and independent Batch
review.

## Dependencies

PF-B2 through PF-B9 as applicable; especially PF-B1-S03 and PF-B7-S02.

## Stop/go conditions

Stop on any blocked high-risk security/runtime evidence, architecture-specific
drift or context failure misreported as product PASS.

## Rollback

Keep the prior image/configuration as a recoverable artifact; stop new rollout
without deleting the persistent volume or migration history.

## Terminal evidence

Accepted implementation candidate:
`6cdfed1179d4f0dbc5266991ad6074741ef7dd75`.

- Independent engineering review: **PASS** —
  [`pf-b10-s01-independent-review.md`](../../docs/implementation/audits/pf-b10-s01-independent-review.md).
- Independent tester acceptance: **PASS** —
  [`pf-b10-s01-independent-acceptance.md`](../../docs/implementation/audits/pf-b10-s01-independent-acceptance.md).
- Exact-candidate local validation: `python scripts/validate.py run full/local`
  on the accepted candidate; format, architecture, security, migration,
  frontend, quality, unit, integration, repository-wide race and AMD64/ARM64
  container profiles all **PASS**.
- Dual-architecture hardened container smoke passed with read-only root,
  dropped capabilities, no-new-privileges, bounded `/tmp`, persistent data,
  `/healthz` 200 and `/readyz` 200.
- Existing WAL, ordered migration ledger and `db.SetMaxOpenConns(1)` policy
  remain unchanged. No CTRL/IMS code, data, connector, product role,
  production record or authority cutover was changed.

Detailed reconciliation:
[`pf-b10-s01-runtime-hardening.md`](../../docs/implementation/audits/pf-b10-s01-runtime-hardening.md).

PF-B10-S01 is **PASS / terminal**. PF-B10-S02 may be promoted only after this
closeout is published.
