# PD-D8-S02 WP-PD8S02-04 — Compatibility retirement evidence

**Date:** 2026-09-26  
**Decision:** Retire only the two IMS package-private compatibility aliases
proven to have zero references. Keep every externally reachable route, local
writer, schema column, projection adapter and feature flag in place.

## Exact candidates

| Repository | Candidate | Purpose |
| --- | --- | --- |
| Platform | `3115caad66b4086535720a03ccf08c2766666edd` | unchanged authority source |
| CTRL | `677dd0dc5573b0bb7ac6c42d2d20c907505e6683` | unchanged guarded consumer source |
| IMS before retirement | `a6c73c1200d73885358056d378e4cff196f467f4` | exact source used by the compatibility ledger |
| IMS retirement candidate | `5f163633c358c636376ebd5ade659a0e7bdf78ee` | removal of two zero-use aliases |

## Retired code

| Item | Zero-use evidence | Safety evidence | Rollback |
| --- | --- | --- | --- |
| `workspaceMemberDiscoveryQueryFromRequest` in IMS `internal/platform/http/member_discovery.go` | Repository-wide `rg` over `internal/**/*.go` found no production or test references other than the definition before removal. The canonical `organisationMemberDiscoveryQueryFromRequest` remains the active implementation. | Unexported package-private function; no route registration or exported interface referenced it. Full IMS HTTP and repository-wide Go suites passed after removal. | Revert commit `5f163633c358c636376ebd5ade659a0e7bdf78ee`. |
| `programmeWorkspaceMemberDiscoveryURL` in the same file | Repository-wide `rg` found no production or test references other than the definition before removal. The canonical `programmeOrganisationMemberDiscoveryURL` remains available. | Unexported package-private wrapper; no route or interface depended on it. Full IMS HTTP and repository-wide Go suites passed after removal. | Revert commit `5f163633c358c636376ebd5ade659a0e7bdf78ee`. |

## Preservation checks

- No SQLite migration, table, column, local ID, public ID, historical actor
  reference, audit row or rollback receipt was changed.
- The active `workspaceMemberDiscoveryURL` compatibility adapter remains
  because `workspace_project_roles.go` still calls it for legacy
  `/workspace/*` bookmarks/forms.
- All CTRL and Platform shared writers, auth paths, projection adapters and
  feature flags remain unchanged and guarded as recorded in the usage ledger.
- The retirement is reversible by one Git revert; no destructive database
  migration was needed.

## Validation

- Static zero-use scan for both retired symbols: **PASS**.
- IMS focused HTTP suite:
  `go test ./internal/platform/http -count=1 -timeout=10m`: **PASS**.
- IMS repository suite:
  `go test ./... -count=1 -timeout=25m`: **PASS**.
- `git diff --check`: **PASS**.
- No unresolved retirement exceptions.

**Verdict:** **PASS / terminal for this bounded retirement.** The active
compatibility ledger is updated with the retirement receipt. Further route,
writer, schema or flag removal remains separately gated by deployment usage
evidence and rollback rehearsal.

