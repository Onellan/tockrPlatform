# PF-B8-S01 independent engineering review

Slice: [`plan/active/pf-b8-s01-platform-shell.md`](../../../plan/active/pf-b8-s01-platform-shell.md)

Implementation candidate: `57b1669312d9336e7f5a0d0812e9135c38e75994`

Review mode: read-only review of the exact implementation candidate before
tester acceptance. No implementation changes were made by this review.

## Review ledger

| Review area | Evidence and conclusion | Verdict |
| --- | --- | --- |
| Authorised scope | The candidate adds the Platform shell, server-rendered Organisation/Workspace selectors, access-gated product launcher page, source-owned templ layouts/primitives/components/pages, and generated Tailwind CSS. PF-B8-S02 administration pages are not implemented. | PASS |
| Platform authority | UI view models receive Platform-owned User, Organisation, Workspace and effective product-access facts. The templates contain no product role, billing, Project or CTRL/IMS operational authority. | PASS |
| Server-side access | Organisation enumeration is scoped to active membership or system administration. Product cards and `/launch/{productKey}` require active User, Organisation membership, entitlement, assignment and Workspace access. Tampered context is returned as least-knowledge `404`. | PASS |
| No-JavaScript path | Context selection uses ordinary GET forms; navigation, account security and product-access pages are server-rendered links/forms. JavaScript is not required or loaded by the shell. | PASS |
| Presentation architecture | `web/layouts`, `web/primitives`, `web/components`, `web/pages`, Tailwind source/generated CSS and Platform semantic CSS follow the approved templ/server-rendered structure. The existing account/security pages now use the shared shell without nested document landmarks. | PASS |
| Accessibility/responsive structure | Semantic headings, labels, native select/details controls, skip link, visible focus treatment, reduced-motion handling and narrow-layout structural breakpoint are present. | PASS |
| Data/locality | New read seams are narrow, use parameterised SQL and preserve the existing one-connection SQLite policy. No migration, shared database, identity mapping or history rewrite was added. | PASS |
| Regression evidence | Focused shell/auth tests, templ generation, format, architecture, frontend, security, `go vet` and `git diff --check` passed on the exact candidate. | PASS |

## Findings

The initial review/visual pass identified a responsive presentation finding:
wide context labels and controls did not share the intended field grid. The
finding was repaired in `5d806c00ad5426ce60722584f4b51406ca2c80c5` and the
wide browser journey was rerun. A follow-up narrow-browser pass identified a
redundant blank mobile application header; it was repaired in
`57b1669312d9336e7f5a0d0812e9135c38e75994` and the narrow journey was rerun.
No R1 findings remain. No R2 finding blocks PF-B8-S01 acceptance.

The product-access page intentionally proves and displays the Platform access
boundary; it does not invent an external consumer URL or assertion transport.
PF-B6 defines the assertion contract while consumer adapters and product-owned
handoff transport remain outside this Slice's authority.

## Review conclusion

**PASS** — the exact final candidate is suitable for independent tester
acceptance.
