# PF-B6-S01 reconciliation evidence

Accepted implementation candidate: `bf3b134e62155481cc98aad7b3613ccdc94129bd`

Terminal closeout candidate: `cfe24a25e663147f07f8c1d5d5e8fe1e17246477`

Plan: [`plan/completed/pf-b6-s01-assertions.md`](../../../plan/completed/pf-b6-s01-assertions.md)

| Acceptance surface | Evidence | Result |
| --- | --- | --- |
| Contract shape | Version 1 compact signed form with strict header/payload allow-lists and the nine approved shared claims | PASS |
| Claim boundary | `usr_`, `org_`, `wsp_` scope only; product roles, billing, passwords, session tokens and entitlement detail are excluded | PASS |
| Access-gated issuance | HTTP issuance calls the existing central effective-access predicate before signing | PASS |
| Cryptographic verification | Ed25519 signature, issuer, audience, version, key ID and algorithm are checked; unknown and malformed values fail closed | PASS |
| Lifetime and replay | Default two-minute lifetime, fifteen-minute maximum, future/expired denial and assertion-ID replay detection are tested | PASS |
| Key rotation | Active plus retained public keys verify during overlap; public-key-only consumer construction is tested | PASS |
| Secure configuration | Startup requires explicit issuer, active key ID, private key and audience allow-list; secret scan passes | PASS |
| HTTP boundary | CSRF, safe unauthorized denial, public-key-only disclosure and no prohibited response fields are tested | PASS |
| Independent engineering review | Read-only review of repaired exact candidate `bf3b134` | PASS; no R1 findings |
| Independent tester acceptance | Separate acceptance run on exact candidate `bf3b134` | PASS |
| Exact-candidate local validation | Implementation candidate profiles plus `full/local` on closeout candidate `cfe24a25` | PASS |
| Container profiles | AMD64 and ARM64 container builds | NOT_APPLICABLE; no authorised Dockerfile exists |

The initial one-connection SQLite policy remains unchanged. No CTRL/IMS route,
code, data, product role, migration or authority cutover was changed. The
first review identified and blocked a private-key-only consumer constructor;
the repaired candidate added `NewVerifier` and was independently re-reviewed
and retested.
