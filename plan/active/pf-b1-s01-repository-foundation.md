# PF-B1-S01 — Repository, standards, agents and validation foundation

Status: **Ready**

## Objective

Make Platform a repository-native delivery workspace with the merged current
CTRL/IMS skills, agents, three-lane workflow and local validation authority.

## Authority and current evidence

Authority is the PF brief, `docs/technical/engineering-workflow.md` and
`docs/technical/testing-execution-contract.md`. Evidence is the source matrix
at the two exact current-main SHAs in `source-alignment-matrix.md`; Platform
started as README-only at `55301d81`.

## Affected files/packages

`.agents/skills/`, `.codex/agents/`, `.codex/reasoning-routing-policy.json`,
`docs/technical/`, `scripts/validate.py`, `scripts/validation_registry.py`,
README and `.gitignore`. No application package.

## Ordered work

1. Merge and deduplicate current CTRL/IMS capability files and preserve the
### WP01 - Ordered work package
   strongest common versions.
Route: kind=other; risk=H[DOC,DEP]
2. Adapt names, paths and authority boundaries to Platform and record the merge
### WP02 - Ordered work package
   policy.
Route: kind=other; risk=H[AUTH,DOC]
3. Register format, unit, integration, architecture, security, migration, race,
### WP03 - Ordered work package
   frontend, AMD64, ARM64, quality and full/local profiles.
Route: kind=other; risk=H[OPS,DOC]

## Migration impact

None. This Slice changes repository control-plane files only and must not
create `platform.db` or a schema migration.

## Security impact

Protect agent configuration and local validation from secret logging. Keep
runtime and product authority out of skills; classify unavailable runtime
profiles as `NOT_APPLICABLE` or context failure, never as PASS.

## Acceptance criteria

- merged capability names and inheritance decisions are recorded;
- three lanes, independent review/tester gates and Batch certification are
  executable workflow rules;
- local profiles resolve through one command authority;
- no GitHub Action is required to validate a local implementation;
- no Platform runtime or CTRL/IMS migration is introduced.

## Tests and evidence

Run `python scripts/validate.py list`, `describe full/local` and `run full/local`;
parse all JSON/TOML/YAML control files; inspect `git diff --check`; confirm no
Go, templ, Docker or database files are present.

## Dependencies

None. This is the first Ready Slice.

## Stop/go conditions

Stop if source files cannot be tied to an exact fetched SHA, if a duplicate
cannot be resolved without losing a gate, or if a profile would claim runtime
proof without its prerequisite. Go only with direct local output.

## Rollback

Revert the foundation commit as one recoverable Git change; no runtime data or
external product system is touched.
