# PF-B8-S01 — Platform layouts, selectors and launcher

Status: Planned

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
