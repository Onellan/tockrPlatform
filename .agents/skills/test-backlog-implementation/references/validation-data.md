# Data and Migration Acceptance Validation

Load for `DATA` test surfaces: schema, migrations, persisted semantics, constraints, nullable values, or data-preservation acceptance.

Use disposable databases outside user-owned/live data or repository-provided disposable fixtures. Verify only the acceptance-relevant cases, including as applicable:

- fresh database reaches the intended schema;
- representative supported pre-change database upgrades without data loss;
- legacy unknown facts remain unknown and unsupported history is not fabricated;
- constraints, foreign keys, uniqueness and indexes enforce the required invariant;
- `NULL`, zero and positive numeric semantics remain distinct;
- failure/rollback leaves no damaging partial persisted state.

Never infer migration correctness from DDL inspection alone when the acceptance condition requires upgrade behavior. Never run migration tests against a live or user-owned database.