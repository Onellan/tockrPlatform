# Performance Acceptance Validation

Load for `PERF` surfaces when acceptance concerns cost shape, responsiveness, throughput, query behavior, boundedness or a performance regression/fix.

Verify correctness first, then the acceptance-relevant performance property:

- establish expected cardinality/growth assumptions;
- inspect for N+1 queries, repeated scans, missing supporting indexes, unbounded memory/result growth or excessive transaction duration where relevant;
- use representative-volume tests, query plans, benchmarks or profiles only when they prove the authoritative condition;
- compare against a reproducible baseline when claiming an improvement or regression fix;
- do not convert a bounded ordinary path into a broad optimization exercise.

Report measured evidence and limits. Do not infer performance from code shape alone.