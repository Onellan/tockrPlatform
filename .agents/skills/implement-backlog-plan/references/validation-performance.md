# Performance Validation

Read when risk includes `PERF` or a materially growing query/collection/export/loop path changes.

- state the expected cardinality/growth dimension and any required bound/pagination;
- inspect for N+1 queries, missing supporting indexes, repeated full scans, unbounded memory growth and long transaction duration;
- consider algorithmic complexity where input size grows materially;
- use `EXPLAIN`, representative-volume tests, benchmarks or profiles only when the risk profile warrants them;
- compare with a reproducible baseline when claiming a performance improvement or regression fix.

Do not benchmark tiny bounded paths merely to create performance evidence.