# PF-B11-S01-R1 — Seed provenance contract correction

Date: 2026-09-17  
Repository: `Onellan/tockrPlatform`  
Implementation candidate: `25c2502298b030f77e38aa246822611f875ad57a`

Status: **PASS / Implemented candidate**

The candidate adds the superseding `platform.read-authority.v2` contract and
compatibility matrix while preserving the terminal v1 event-only contract and
the unchanged `platform-events-v1` payload allow-list. The shared record seam
now validates an explicit v2 `event` or `migration_seed` provenance union.

Migration seeds carry only positive migration version, bounded migration name
and SHA-256 checksum. Event records carry only `evt_*`, positive aggregate
sequence and source schema version. Mixed provenance, malformed checksums,
fabricated event identity and unknown provenance kinds fail closed.

No PF-B11-S02 snapshot persistence, feed, HTTP route, CTRL/IMS runtime code,
product-role, billing or authority-cutover behavior was implemented.

The Platform quality validator was updated from the previous fixed 26-slice
inventory to the authorized 27-slice inventory so the new R1 plan is checked
exactly rather than hidden by a minimum-count rule.

Evidence on this candidate:

- v2 contract/unit tests: **PASS**;
- integration, migration and race profiles: **PASS**;
- format, quality, architecture and security profiles: **PASS**;
- independent engineering review: **PASS**;
- independent tester acceptance: **PASS**.
