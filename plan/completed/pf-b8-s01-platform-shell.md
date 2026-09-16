# PF-B8-S01 — Platform layouts, selectors and launcher

Status: **Implemented / terminal**

## Objective

Build the shared Platform shell and initial navigation surfaces: sign-in/account
security, Organisation selector, Workspace selector and product launcher.

## Authority and current evidence

Authority is the Platform presentation contract and initial UI scope. CTRL/IMS
PB establishes the common templ/Tailwind/shadcn-templ direction; their product
navigation is not copied as Platform behavior.

## Affected files/packages

`web/layouts`, `web/primitives`, `web/components`, `web/pages`, Tailwind
generated CSS, HTTP route/view models and progressive-enhancement JS.

## Ordered work

1. Define UI routing signature, semantic tokens and accessible layout/component
### WP01 - Ordered work package
   seams for Operate surfaces.
Route: kind=ui; risk=H[UI,AUTH,API]
2. Implement server-rendered shell, selectors and launcher with no-JS fallback.
### WP02 - Ordered work package

Route: kind=ui; risk=H[UI,AUTH,API]
3. Prove responsive, focus, error, empty, denied and loading states with real
### WP03 - Ordered work package
   browser evidence where available.
Route: kind=ui; risk=H[UI,AUTH,PERF]

## Migration impact

No product UI migration. Platform starts with new templates and does not import
CTRL/IMS operational screens.

## Security impact

Views render already-authorized state; selector/launcher endpoints re-check
scope and CSRF. JS cannot grant access.

## Acceptance criteria

Core navigation/forms work without JavaScript, visual language aligns with
CTRL/IMS contract, unauthorized products/Workspaces are absent and direct URL
access is denied server-side.

## Tests and evidence

Frontend profile, HTTP auth/CSRF tests, templ generation/build, keyboard/focus
and browser verification at wide/narrow states.

## Dependencies

PF-B6-S02, PF-B7-S02 and PF-B1-S03.

## Stop/go conditions

Stop if UI requires React/Vue/Svelte, runtime Node, client-side authorization or
product operational screens.

## Rollback

Revert routes/templates while retaining contract docs; no access data changes.

## Terminal evidence

Accepted implementation candidate: `57b1669312d9336e7f5a0d0812e9135c38e75994`.

- Independent engineering review: **PASS**; the initial responsive findings
  were repaired sequentially in `5d806c00ad5426ce60722584f4b51406ca2c80c5`
  and this final candidate, then the affected browser journeys were rerun.
- Independent tester acceptance: **PASS** for shared shell, no-JavaScript
  navigation/forms, authorized context/product visibility, fail-closed direct
  URLs, responsive/accessibility structure and Platform boundary.
- Exact-candidate local validation: format, architecture, security, frontend,
  quality, unit and integration children **PASS**. The repository composite
  `full/local` race child exceeded its fixed 300-second bound and is recorded
  as **TIMEOUT**, not PASS; the extended changed-package race command passed.
- Browser evidence: signed-in wide and narrow launcher journeys passed
  semantic snapshot and visual inspection. The only console observation was a
  non-blocking missing `/favicon.ico` request; no application error occurred.
- Container build profiles: **NOT_APPLICABLE** because no authorised
  Dockerfile exists.
- The initial one-connection SQLite policy remains unchanged. No CTRL/IMS
  production code, data, migration, product role or authority cutover was
  changed.

Detailed evidence:
[`reconciliation`](../../docs/implementation/audits/pf-b8-s01-platform-shell.md),
[`independent review`](../../docs/implementation/audits/pf-b8-s01-independent-review.md),
[`independent acceptance`](../../docs/implementation/audits/pf-b8-s01-independent-acceptance.md).
