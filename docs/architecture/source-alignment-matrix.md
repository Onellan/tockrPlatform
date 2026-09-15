# CTRL / IMS current-main alignment matrix

This is the evidence baseline for Priority PF. It was refreshed from the
remote `origin/main` refs on 2026-09-15; local branches were not treated as
authoritative because both product checkouts were behind their remotes.

## Exact source SHAs

| Repository | Ref used | Exact SHA | Role |
| --- | --- | --- | --- |
| `Onellan/tockrctrl` | `origin/main` | `47d20f29ad36e59d44b1154ebcde1ed5a99905f6` | CTRL architecture and product boundary reference |
| `Onellan/tockrims` | `origin/main` | `aa5de6c4672114c35b3f31fdc3177189a78c9b5d` | IMS architecture and workflow reference |
| `Onellan/tockrPlatform` | `origin/main` before foundation | `55301d8103feee7c0c919703ae01457af0980dcd` | Platform starting point |

## Verified evidence

| Concern | CTRL current-main evidence | IMS current-main evidence | PF conclusion |
| --- | --- | --- | --- |
| Backend | `architecture.md`, `docs/technical/coding-standards.md` | `architecture.md`, `docs/technical/coding-standards.md` | Go modular monolith, Chi, `modernc.org/sqlite`, WAL, capability-local persistence |
| Presentation | `docs/technical/presentation-architecture-contract.md`, PA/PB records | `docs/technical/presentation-architecture-contract.md`, PB records | templ, source-owned shadcn-templ, Tailwind, server-rendered default, bounded JS |
| Domain locality | `internal/store`, `internal/db/sqlite`, platform structural contract | `internal/domain`, `internal/store`, `internal/db/sqlite`, PA/PD records | narrow capability contracts; no shared product database |
| Authentication/session | `internal/auth`, session projection and security standards | `internal/auth`, `internal/db/sqlite/session_hot_path.go`, session hot-path standard | DB-backed revocation, credential-free request projection, secure sessions and CSRF |
| Organisation/Workspace | PA structural contract and organisation/workspace stores | PA slices, workspace scope context and stores | Platform becomes shared authority; product roles remain local |
| SQLite/migrations | ordered named `schema_migrations`, fresh and upgrade evidence | exact-prefix ledger, transaction-per-migration, fail-closed divergence | `platform.db`, one ordered named ledger, fresh/upgrade/reopen proof |
| Runtime | bounded HTTP resources, AMD64/ARM64, hardened container | same plus non-root/container contract | `/healthz`, `/readyz`, non-root, dropped capabilities, read-only root, persistent volume |
| Delivery | three-lane workflow, independent gates, local validation | workflow and validation execution contract | merge current shared workflow; local validator is authority; no Actions-only validation |

## Conflicts and decisions

1. **SQLite pool width:** CTRL describes a single open connection while IMS
   records a measured file-backed `4/2` pool after serialized migrations. This
   is a real implementation semantic conflict. PF freezes WAL, serialized
   migration startup and the single-instance boundary, but deliberately does
   not choose a pool width; PF-B1-S03 must make that decision from Platform
   measurements before runtime code.
2. **Presentation incumbent:** CTRL has templ-based migration inventory while
   IMS still has legacy `html/template` inventory. Both current-main contracts
   converge on real templ layouts/components and Tailwind. Platform adopts the
   target contract and does not copy either incumbent.
3. **Identity continuity:** neither repository’s existing public IDs are
   assumed to identify the same people or tenants. PF uses stable opaque
   `usr_`, `org_` and `wsp_` IDs and requires explicit reconciliation mapping.
4. **Product roles:** CTRL and IMS contain product-specific role/scoping
   concepts. PF does not merge them; only Platform roles and shared access
   predicates are canonical here.
5. **Validation authority:** CTRL’s richer agent-evaluation/release material
   and IMS’s newer central validation/test-execution material are both retained
   in the merged capability set. Platform’s local validator is the only command
   authority for Platform evidence.

No third interpretation is silently substituted for an unresolved source
semantic. The affected Slice is the authority for resolving it before code.
