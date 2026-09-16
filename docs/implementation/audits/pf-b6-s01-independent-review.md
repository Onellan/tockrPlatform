# PF-B6-S01 independent engineering review

Candidate reviewed: `bf3b134e62155481cc98aad7b3613ccdc94129bd`

Verdict: **PASS**. No R1 finding remains.

The read-only review verified that:

- the assertion payload is a strict versioned allow-list matching the Platform
  contract and contains no product role, billing, credential, session or full
  entitlement data;
- issuance is behind the existing active effective-access proof and returns
  only shared `usr_`, `org_` and `wsp_` scope;
- Ed25519 signatures, issuer, audience, version, key ID, expiry, future time,
  scope shape and replay identity are fail-closed;
- key rotation retains configured public verification keys and consumers can
  construct a verifier without private signing material;
- startup configuration errors do not print key material, while the public-key
  endpoint exposes public data only;
- HTTP issuance requires session CSRF and maps unauthorized access safely; and
- no CTRL/IMS authority, product role, billing or SQLite policy was changed.

The review began from the repaired candidate after the initial private-key-only
consumer finding was fixed. No later documentation or plan change is included
in the reviewed implementation candidate.
