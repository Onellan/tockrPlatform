# Platform frontend toolchain

Platform follows the CTRL/IMS presentation direction without importing their
runtime. `templ` generates typed server-rendered Go components and the pinned
Tailwind standalone CLI generates the shared presentation layer. No Node.js,
npm or frontend runtime is required by the Platform process.

| Tool | Pin | Purpose |
| --- | --- | --- |
| `templ` | `v0.3.1020` | Generate typed Go output from `.templ` sources |
| Tailwind standalone CLI | `v4.3.3` | Generate deterministic CSS without Node.js/npm |

The Windows wrapper is [`scripts/frontend-tools.ps1`](../../scripts/frontend-tools.ps1).
It verifies the Tailwind SHA-256 before execution, generates disposable CSS for
`check`, and writes `web/static/presentation.css` only for `generate-runtime`.
The runtime serves the generated asset from the existing static directory; it
does not execute a frontend tool.

The source-owned layers are:

```text
web/layouts       document and application shells
web/primitives    reusable structural primitives
web/components    Platform semantic components
web/pages         server-rendered view models and pages
web/static        Tailwind source, generated CSS and semantic Platform CSS
```

The shell and selectors work as normal HTML forms and links without
JavaScript. JavaScript is not authoritative for access, CSRF or product
handoff. Platform only displays products returned by the complete shared
access predicate; CTRL/IMS roles and operational records remain consumer-owned.

Run on Windows:

```text
pwsh scripts/frontend-tools.ps1 check
pwsh scripts/frontend-tools.ps1 generate-runtime
```
