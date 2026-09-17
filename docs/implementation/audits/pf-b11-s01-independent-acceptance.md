# PF-B11-S01 independent tester acceptance

Accepted candidate: `b79b9321a06dd1c0e25381127dc61e861bae520d`

The tester independently derived the acceptance rows from the terminal
PF-B11-S01 authority and tested only the S01 contract surface. Commands were
resolved through the repository validation authority; no later Slice runtime
was exercised or inferred.

| Acceptance row | Evidence | TestContext / result |
| --- | --- | --- |
| S01-AC01 — separate version and complete allow-list/forbidden fields | S01-TEST-E01, S01-TEST-E02 | `unit` contract fixtures plus exact contract docs; PASS |
| S01-AC02 — bounded bootstrap/incremental semantics fail closed for non-current state | S01-TEST-E01 | Unit state, request-bound and route fixtures plus exact state/cursor/checksum docs; PASS |
| S01-AC03 — explicit rotated/replay-resistant machine authentication independent of browser assertions | S01-TEST-E01, S01-TEST-E02 | Canonical request/digest negative fixtures, auth contract and security profile; PASS |
| S01-AC04 — same protocol for CTRL and IMS with only consumer/product pairing | S01-TEST-E01 | Consumer/product matrix fixtures and compatibility contract; PASS |
| S01-AC05 — Staff Engineer plan review and implementation review resolve material decisions | S01-TEST-E03 | Staff Engineer planning review PASS TO IMPLEMENTATION plus repeated independent engineering review PASS; PASS |

## Evidence ledger

### S01-TEST-E01

```text
TestContext:
  candidate=b79b9321a06dd1c0e25381127dc61e861bae520d
  authority=PF-B11-S01-AC01..AC04
  surface=API|AUTH|DEP|DOC
  profile=unit
  command_source=scripts/validation_registry.py and scripts/validate.py
  preflight=PASS
```

`python scripts/validate.py run unit` passed. The repository-resolved command
included `./internal/platform/readauthority`; version mismatch, consumer
matrix, unknown-field, forbidden-record, state, canonical-request and route
fixtures passed.

### S01-TEST-E02

```text
TestContext:
  candidate=b79b9321a06dd1c0e25381127dc61e861bae520d
  authority=PF-B11-S01-AC01..AC04
  surface=API|AUTH|DEP|DOC
  profile=format,architecture,security
  command_source=scripts/validation_registry.py and scripts/validate.py
  preflight=PASS
```

All three named profiles passed with no missing authority files, unformatted
Go files or secret-pattern hits.

### S01-TEST-E03

```text
TestContext:
  candidate=b79b9321a06dd1c0e25381127dc61e861bae520d
  authority=PF-B11-S01-AC05
  surface=GOV|DOC|DEP
  profile=plan-routing
  command_source=scripts/validate_plan_routing.py
  preflight=PASS
```

The S01 plan has three valid route signatures and its routing validator passed.

The earlier unsupported `python scripts/validate.py go-test ...` invocation is
recorded as `INVOCATION_FAIL`, not as a product failure or acceptance result.
No required S01 evidence is BLOCKED / NOT RUN.

**Independent tester result: PASS.** This acceptance does not certify S02,
S03, S04 or the PF-B11 Batch.
