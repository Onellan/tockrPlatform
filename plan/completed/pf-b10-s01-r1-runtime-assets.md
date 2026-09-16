# PF-B10-S01-R1 — Runtime asset packaging repair

Status: **Implemented / terminal**

## Objective

Repair the PF-B10-S01 container packaging regression found by the PF-B10-S02
real-browser gate: the distroless image must serve Platform static assets with
their correct MIME types and the public login route must not produce a missing
favicon console error.

## Authority and current evidence

This was a narrow repair loop for the terminal PF-B10-S01 runtime-hardening
scope. Authority was the security/runtime contract, the PF-B10-S01 plan and
the browser evidence that the accepted image omitted `web/static`.

## Affected files/packages

Dockerfile image contents, the public favicon route, runtime packaging tests,
container smoke evidence and candidate-bound reconciliation records.

## Ordered work

1. Package the source-owned static assets in the final distroless image and
### WP01 - Ordered work package
   provide a safe no-content favicon response.
Route: kind=other; risk=H[DEPLOY,UI,OPS]
2. Add focused regression coverage for the public asset boundary.
### WP02 - Ordered work package
Route: kind=other; risk=H[UI,API]
3. Rebuild AMD64/ARM64 images, run real-browser `/login` and
### WP03 - Ordered work package
   health/readiness checks, then rerun the exact-candidate local profile.
Route: kind=other; risk=H[DEPLOY,UI,AUTH]

## Migration impact

None. No database schema, migration ledger, persistent data or SQLite policy
changed.

## Security impact

The non-root, read-only-root, dropped-capability, bounded-tmpfs,
persistent-volume and secret-free image contract remains unchanged. No shell,
runtime Node dependency or client-side authority was added.

## Acceptance criteria

The final image serves `/static/*.css` as CSS from both architecture targets;
`/login` renders with no asset MIME/404 console errors; `/healthz` and
`/readyz` remain safe and operational; and all existing S01 gates remain green.

## Dependencies

PF-B10-S01 terminal historical candidate; PF-B10-S02 was paused until this
repair closed.

## Terminal evidence

Accepted repair candidate:
`2dbac87909b296e66da33f1e9f26d049bbbd1bd7`.

- Independent engineering review: **PASS** —
  [`pf-b10-s01-r1-independent-review.md`](../../docs/implementation/audits/pf-b10-s01-r1-independent-review.md).
- Independent tester acceptance: **PASS** —
  [`pf-b10-s01-r1-independent-acceptance.md`](../../docs/implementation/audits/pf-b10-s01-r1-independent-acceptance.md).
- Exact-candidate local validation: `python scripts/validate.py run full/local`,
  with architecture, security, quality, audit, routing, race and AMD64/ARM64
  container profiles all **PASS**.
- Real-browser `/login`, `/healthz` and `/readyz` checks pass with no console
  errors; CSS is `200 text/css` and favicon is safe `204`.
- No migration, SQLite connection-policy, CTRL/IMS code/data/connector,
  product-role or authority-cutover change was introduced.

Detailed reconciliation:
[`pf-b10-s01-r1-runtime-assets.md`](../../docs/implementation/audits/pf-b10-s01-r1-runtime-assets.md).

PF-B10-S01-R1 is **PASS / terminal**. PF-B10-S02 may now be re-promoted.
