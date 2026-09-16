# PF-B6-S01 independent tester acceptance

Candidate: `bf3b134e62155481cc98aad7b3613ccdc94129bd`

Verdict: **PASS**.

| AC | Independent evidence | Result |
| --- | --- | --- |
| AC01 — approved assertion claims | Strict payload/header contract and prohibited-field checks | PASS |
| AC02 — access-gated issuance | HTTP test proves assignment/entitlement/Workspace access is required before issuance | PASS |
| AC03 — cryptographic and semantic verification | Signature, issuer, audience, version, scope, key and algorithm negative checks | PASS |
| AC04 — bounded validity | Default/max lifetime, expired and future assertion checks | PASS |
| AC05 — replay and rotation | Replay denial, retained-key overlap and public-key-only consumer verifier | PASS |
| AC06 — secure boundary | Explicit startup configuration, CSRF, safe denial, public-key disclosure and secret scan | PASS |

Independent exact-candidate evidence passed through repository profiles:
`format`, `architecture`, `security`, `quality`, `unit`, `integration` and
repository-wide `race`. The assertion package and the exact HTTP assertion
selector were also run uncached against this candidate. Container builds are
**NOT_APPLICABLE** because no authorised Dockerfile exists.

No later Slice was tested or accepted.
