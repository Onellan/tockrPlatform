# Platform SQLite migration contract

The first runtime database is `platform.db`. SQLite schema evolution is owned
by one ordered named `schema_migrations` ledger.

## Ledger rules

- each migration has a positive ordered version and stable name;
- applied rows must be an exact supported prefix;
- renamed, missing, reordered or unknown rows fail closed;
- each migration and ledger row commit in one transaction;
- a failed migration leaves the previous valid prefix intact;
- startup never resets, recreates or discards an existing database.

## Required evidence

Every schema-bearing Slice must prove:

1. fresh database creation;
2. upgrade from a representative prior database;
3. close and reopen with the ledger intact;
4. divergence rejection;
5. preservation of historical facts and known provenance.

No migration may fabricate actors, timestamps, reasons, membership decisions or
identity matches. CTRL and IMS databases are source evidence for future
reconciliation, not Platform databases to be rewritten in place.
