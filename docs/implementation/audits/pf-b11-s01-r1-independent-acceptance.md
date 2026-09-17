# PF-B11-S01-R1 — Independent tester acceptance

Date: 2026-09-17  
Repository: `Onellan/tockrPlatform`  
Accepted candidate: `25c2502298b030f77e38aa246822611f875ad57a`

Verdict: **PASS**

TestContext:

```text
candidate=25c2502298b030f77e38aa246822611f875ad57a
authority=PF-B11-S01-R1-AC01..AC06
surfaces=API|DATA|HIST|AUTH|DEP|DOC
command_source=scripts/validation_registry.py via scripts/validate.py
preflight=PASS
```

| Acceptance condition | Evidence | Result |
| --- | --- | --- |
| R1-AC01: v1 remains event-only | `unit`: `TestV1RemainsEventOnly`; terminal v1 documents unchanged | PASS |
| R1-AC02: v2 is an exact contract artifact | v2 contract and compatibility documents; `format`, `quality`, `architecture` | PASS |
| R1-AC03: migration seed is explicit and exclusive | `unit`: migration-seed acceptance, malformed checksum, mixed/fabricated provenance and wire-shape tests | PASS |
| R1-AC04: v1 event feed remains unchanged | event validation and v1 contract diff review; no event allow-list or `internal/events` change | PASS |
| R1-AC05: CTRL/IMS handoff is aligned | current CTRL/IMS PD-D5 plans require v2 and seed preservation while remaining gated on terminal PF-B11 | PASS |
| R1-AC06: candidate validation is green | `format`, `quality`, `architecture`, `security`, `unit`, `integration`, `migration` and `race` | PASS |

No required evidence was blocked or not run. This acceptance covers only the
R1 contract correction; it does not accept PF-B11-S02 runtime behavior.
