# Validation Reference Index

Detailed validation is split so the implementer reads only affected surfaces.

- [Core validation](validation-core.md): always; focused evidence, Git diff/worktree checks, Go-wide checks when Go changes, docs check when Markdown changes.
- [Persistence validation](validation-persistence.md): `DATA`, `HIST`, `CONC`.
- [UI validation](validation-ui.md): `UI`.
- [Security validation](validation-security.md): `AUTH`.
- [Performance validation](validation-performance.md): `PERF`.
- [Runtime and dependency validation](validation-runtime-dependencies.md): `OPS`, `DEPLOY`, `DEP`.

Read only the triggered references. High risk means deeper targeted evidence, not loading every category. Never claim a skipped or unavailable check passed.