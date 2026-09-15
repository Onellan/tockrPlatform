# Persistence Validation

Read when risk includes `DATA`, `HIST` or `CONC`, or when schema/migrations/persisted semantics materially change.

- prove fresh database creation;
- prove a representative supported upgrade path;
- verify changed constraints, foreign keys, uniqueness/index behavior and `NULL`/zero semantics;
- verify all material writers/readers of the changed invariant;
- force and verify rollback when partial writes would be damaging;
- verify lifecycle/state transitions in the mutation transaction when races matter;
- prove stale-write/concurrency behavior where applicable;
- preserve known legacy facts and never fabricate unsupported historical state.

Use disposable databases. Add only the cases relevant to the changed invariant.