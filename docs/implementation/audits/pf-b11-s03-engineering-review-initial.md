# PF-B11-S03 independent engineering review — initial candidate

**Candidate:** `d466f9a` (`d466f9a` full SHA recorded in the acceptance ledger)
**Decision:** **R1 — repair required**
**Authority:** PF-B11-S03 WP01–WP04 and S03-AC01–S03-AC06

## Finding

The machine-authentication timestamp verifier calculated the absolute
difference using signed `int64` subtraction. A parseable extreme future Unix
timestamp could overflow that subtraction and satisfy the skew comparison.
That violated S03-AC01's bounded timestamp requirement and required repair
before tester acceptance.

## Evidence

The rest of the fixed candidate review covered the v2 signed-request boundary,
deployment-managed consumer/key binding and overlap, durable nonce replay
protection, bounded body/page/response/rate limits, snapshot and feed route
seams, source-gap detection, safe errors, and Platform/CTRL/IMS boundary
preservation. The identified timestamp defect was the only blocking review
finding.

