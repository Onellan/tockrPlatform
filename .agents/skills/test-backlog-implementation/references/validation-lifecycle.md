# Lifecycle, History and Concurrency Acceptance Validation

Load for `HIST` or `CONC` surfaces: governed lifecycle, immutable history/audit, stale writes, duplicate transitions, concurrency, idempotency or transactional integrity.

Verify only acceptance-relevant lifecycle/integrity behavior, including as applicable:

- valid and forbidden transitions;
- stale version/write rejection;
- duplicate submission/idempotency behavior;
- concurrent approvals/revisions/transitions;
- forced transactional failure and exact rollback state;
- immutable history/audit facts and actor/reason/timestamp semantics;
- exact final authoritative state after competing operations.

Do not treat “no error” as sufficient evidence. Assert exact final state/history facts required by authority.

The engineering reviewer owns architecture and transaction-design quality. The tester inspects transaction/writer paths only far enough to identify acceptance risks and construct reproducible evidence.