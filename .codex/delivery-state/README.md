# Compact delivery state

`backlog_delivery` may persist one compact state file per active item at
`.codex/delivery-state/<item>.json` (or the repository's established item
extension). The file is resumable operational state, not product or terminal
history.

Required fields are `schema_version`, `item`, `plan`, `candidate`, `authority`,
`next`, `packages`, `routing`, `review`, `test`, `release`, `metric_attempts`
`evidence_ids`, `finding_ids` and `metric_inflight`. `candidate` binds commit, working-tree fingerprint and
derived ID. `authority` binds the normalized plan path and SHA-256 hashes for
the plan plus any explicitly supplied authority files. `next` preserves the exact phase/package/stage. Completed
packages remain `status: pass` and are never replayed. An orphaned
`metric_inflight` record is retained and resumes the same invocation identity;
it is not silently treated as completed.

Validate or inspect a state file with:

```text
python scripts/delivery_state.py validate .codex/delivery-state/<item>.json
python scripts/delivery_state.py resume .codex/delivery-state/<item>.json
python scripts/evidence_ledger.py <ledger> --acceptance-row AC01 --acceptance-row AC02
```

Before a resume pointer is returned, the current candidate and every authority
hash are recomputed and compared. A stale candidate or authority file is
`BLOCKED / NOT RUN`; it cannot skip a gate.

The state file must contain identifiers, hashes, routing decisions, evidence
IDs, outcomes and minimal failure reproduction context. `evidence_ids` and
`finding_ids` preserve the independent gate/finding identities needed for
resume. A routing decision must
retain the package, kind, risk level/codes, selected effort and runtime. An
orphaned metric invocation must retain its agent, invocation ID and candidate
identity so it cannot be replayed or silently completed. Do not copy specialist
narratives, successful command output or repeated plan/authority prose into it.
