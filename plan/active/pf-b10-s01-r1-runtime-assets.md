# PF-B10-S01-R1 — Runtime asset packaging repair

Status: Ready

## Objective

Repair the PF-B10-S01 container packaging regression found by the PF-B10-S02
real-browser gate: the distroless image must serve Platform static assets with
their correct MIME types and the public login route must not produce a missing
favicon console error.

## Authority and current evidence

This is a narrow repair loop for the terminal PF-B10-S01 runtime-hardening
scope. Authority is the security/runtime contract, the PF-B10-S01 plan and the
browser evidence that the accepted image omitted `web/static`. PF-B10-S02 is
paused until this repair is terminally reviewed, accepted and validated.

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
3. Rebuild AMD64/ARM64 images, run real-browser login/static checks, then rerun
### WP03 - Ordered work package
   the exact-candidate local profile and affected independent gates.
Route: kind=other; risk=H[DEPLOY,UI,AUTH]

## Migration impact

None. No database schema, migration ledger, persistent data or SQLite policy is
changed.

## Security impact

The repair must preserve the non-root, read-only-root, dropped-capability,
bounded-tmpfs, persistent-volume and secret-free image contract. It must not
add a shell, runtime Node dependency or client-side authority.

## Acceptance criteria

The final image serves `/static/*.css` as CSS from both architecture targets;
`/login` renders with no asset MIME/404 console errors; `/healthz` and
`/readyz` remain safe and operational; and all existing S01 gates remain green.

## Tests and evidence

Focused HTTP test, AMD64/ARM64 builds and smoke, real-browser `/login` and
health/readiness checks, full/local validation, architecture/security/quality
profiles and independent repair review/acceptance.

## Dependencies

PF-B10-S01 terminal candidate `6cdfed1179d4f0dbc5266991ad6074741ef7dd75`;
PF-B10-S02 is paused until this repair closes.

## Stop/go conditions

Stop if either architecture omits static assets, the browser reports an asset
MIME/404 error, or any existing runtime/security/boundary gate regresses.

## Rollback

Retain the published PF-B10-S01 image/candidate and revert only the repair
candidate; do not remove the persistent volume or alter migration history.
