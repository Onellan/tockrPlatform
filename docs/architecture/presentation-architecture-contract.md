# Platform presentation architecture contract

Status: foundation planning authority; no Platform UI runtime has been built.

```text
Browser
  ↓
Go + Chi HTTP boundary
  ↓
real templ layouts / pages / components
  ├── source-owned shadcn-templ primitives
  └── Tockr semantic components
  ↓
Tailwind design tokens / generated CSS
  ↓
server-rendered HTML
  + small progressive-enhancement JavaScript
  + HTMX only where a concrete server-rendered fragment interaction justifies it
```

Rules:

- templ is the canonical renderer/component model;
- Tailwind is the canonical styling and token foundation;
- primitives are source-owned rather than a runtime package dependency;
- server-rendered HTML is the default and core forms/navigation work without JS;
- JavaScript is bounded and never authoritative for authorization, CSRF or
  irreversible decisions;
- React, Vue, Svelte, SPA shells and runtime Node.js are out of scope;
- Platform UI visually aligns with CTRL/IMS without importing their runtime;
- authorization and business rules remain server-side.

Initial UI surfaces are sign-in/account security, organisation/workspace
selectors, product launcher, organisation administration, workspace
administration and system product catalogue. CTRL/IMS operational screens are
not Platform UI.
