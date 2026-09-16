# PF-B9-S01 independent tester acceptance

Implementation candidate accepted: `eafb9451572d248275f6eafe6174a547a4eceadb`

The acceptance was executed independently after the final review repairs and
is bound to the exact implementation candidate above. No production source,
Platform database or CTRL/IMS repository was opened by the CLI or tests.

| Acceptance condition | Command/evidence | Result |
| --- | --- | --- |
| Deterministic proposals for normalized Users, Organisations, Workspaces and memberships | `go test -count=1 ./internal/platform/reconciliation ./cmd/platform-reconcile` | PASS |
| Same inputs produce identical output and source IDs do not determine the candidate Platform ID | `TestBuildIsRepeatableAndUsesCanonicalKeysNotSourceIDs`; CLI fixture run twice | PASS |
| Ambiguous matches stop for explicit review | `TestBuildBlocksAmbiguousAndCollidingIdentity` | PASS |
| Source identity collisions remain visible and propagate across merged sources | `TestBuildPropagatesSourceIdentityCollisionAcrossMergedSources` | PASS |
| Every proposal retains exact source/version/source-ID provenance | Cross-source proposal test and fixture CLI report `source_refs` | PASS |
| Canonical match keys are not repeated in the report | CLI fixture output inspection; match-key digest only | PASS |
| Missing or unresolved parent/user identity blocks without a Platform ID | `TestBuildBlocksMissingProvenanceAndUnresolvedDependencies`, `TestBuildBlocksRelationshipWhenSameSourceProvenanceIsMissing` | PASS |
| Read-only operational proof | `go run ./cmd/platform-reconcile -input internal/platform/reconciliation/testdata/inventory.json`; no database/source connector in changed surfaces | PASS |
| Repeatability hash | Two identical CLI outputs; SHA-256 `147b5654b99c879f70822c0ef38b4d6478ba62c5d86d109bb900575ebda3d112` | PASS |
| Focused concurrency evidence | `go test -race -count=1 ./internal/platform/reconciliation` | PASS |
| Repository exact-candidate validation | `python scripts/validate.py run full/local` on candidate above | PASS |
| Supporting repository profiles | `format`, `architecture`, `security`, `quality`, `frontend`, `unit`, `go vet ./...`, `git diff --check` | PASS |
| Container profiles | No authorised Dockerfile exists | NOT_APPLICABLE |
| Platform boundary and SQLite policy | Source review; no CTRL/IMS code/data/role/cutover and no SQLite pool or migration change | PASS |

Verdict: **PASS** for the PF-B9-S01 acceptance criteria. No required gate is
BLOCKED / NOT RUN.
