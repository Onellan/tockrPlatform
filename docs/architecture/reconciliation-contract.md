# Platform reconciliation contract

PF-B9-S01 provides a read-only preparation seam for identity reconciliation.
It does not open CTRL, IMS or Platform databases and it does not import,
rewrite or accept any record.

## Adapter boundary

An adapter supplies normalized source observations with:

- the source name (`ctrl` or `ims`);
- the exact inspected source SHA;
- the source entity and source identity;
- an explicitly authorized, source-neutral `match_key`; and
- parent/user relationship keys for Workspaces and memberships.

The reconciliation engine never derives a match key from a current CTRL/IMS
ID, email, display name or row order. Missing provenance, identity keys or
relationship references remain `blocked`.

## Proposal rules

The engine hashes the entity kind and canonical match key into an opaque,
stable candidate Platform ID. The candidate is only a proposal and always
requires explicit review. Equal canonical keys from CTRL and IMS may share a
proposal when their identity facts agree; conflicting facts are
`ambiguous`, duplicate source identities are `collision`, and unresolved
parent/user mappings are `blocked`.

Reports are timestamp-free and sorted for repeatability. They retain exact
source/version/source-ID provenance while emitting only a SHA-256 digest of
the canonical match key. No actor, time, role decision or historical reason
is fabricated.

## Operational boundary

`cmd/platform-reconcile` accepts fixture/adapter JSON from stdin or a file and
writes a deterministic report to stdout or an explicitly named disposable
output file. It has no database flags and no production source connector.
PF-B9-S02 owns fixture-only import-shaped workflow, checkpoints and rollback;
production import and authority cutover remain outside PF.

## Fixture-only import rehearsal

PF-B9-S02 converts an entirely proposed S01 report into a deterministic
manifest only when every record is resolved. The manifest binds the source
report digest, opaque candidate IDs, source provenance and explicit operator
approval. Ed25519 signing and verification fail closed for tampering or
missing approval; the manifest scope is always `fixture-only`.

The importer is an in-memory disposable rehearsal. It applies records in
manifest order, HMAC-seals each checkpoint, resumes from the next index after
a paused run, and returns an idempotent no-op for a completed manifest.
Compensating rollback removes only records from the verified fixture manifest
and appends an audit event while retaining the signed manifest and audit
history. No Platform SQLite migration or production source connector is
introduced.
