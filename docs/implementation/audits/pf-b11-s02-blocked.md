# PF-B11-S02 blocked authority record

Slice: PF-B11-S02 — Durable snapshot and source cursor

Status: **BLOCKED / NOT RUN**

## Evidence

- The frozen S01 contract requires every read-authority record to contain an
  `evt_*` source event identity, positive aggregate sequence and source schema
  version.
- Platform migration 6 inserts active `product.tockrctrl` and
  `product.tockrims` rows before migration 8 creates `platform_outbox`.
- The terminal `platform-events-v1` allow-list contains only
  `platform.product.retired` for Product and has no `platform.product.created`
  event.
- A fresh or upgraded database therefore contains active Product authority
  rows with no committed v1 source event from which S02 can derive the
  mandatory snapshot provenance.

## Stop decision

S02 was not implemented. A synthetic event ID, omitted Product records or a
mislabelled retirement event would fabricate or misstate source history. A new
Product-created event would change the terminal v1 event payload contract,
which the existing S02 plan explicitly prohibits. This is `PREREQUISITE_FAIL`
at the authority surface and remains **BLOCKED / NOT RUN**, not a product test
failure.

PF-B11-S03 and PF-B11-S04 were not started. No S02 migration, snapshot table,
consumer data, CTRL/IMS code or authority cutover was changed. Resolution
requires explicit superseding authority for the source-provenance contract or
an authorised compatible event-history migration before S02 can be replanned.
