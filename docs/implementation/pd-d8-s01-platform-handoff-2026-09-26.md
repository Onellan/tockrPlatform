# PD-D8-S01 Platform handoff — 2026-09-26

## Certified candidate set

- Platform: `cc78557d8048bac6392f837166cb101460f43c98`
- CTRL published `main`: `71406518624d3260d96f52e0cbde8eb8b89d440c`; accepted code candidate `bfbd808cec5c3dc705bef1c2e8e2e6296dead76c`
- IMS published `main`: `d5d438b8707ae92136e4677bdc2a96ecbc76199c`; accepted code candidate `63e47c0929fd417b6d7c526bb662ba7fdf5cd5a5`
- D7 source handoff remains bound to the signed manifest, restore point and projection cursor recorded in each consumer candidate-freeze JSON.

## Platform-facing certification result

CTRL and IMS independently reviewed and accepted the D8-S01 matrix. Invalid, expired, replayed and revoked assertions; missing/inactive product access; cross-Organisation/Workspace/project or programme tampering; membership revocation; stale/gap/resync/outage states; denied-write no-drift and audit correlation all passed. Platform read authority remains the snapshot/feed source, while ordinary consumer requests use local projections and bounded freshness policy.

The only repair was a tests-only current-time reconciliation fixture in each consumer. No Platform code, schema, authority data, manifest, mapping or production writer changed. The final IMS browser ledger passed all 47 rows on a clean published candidate; CTRL and IMS repository profiles, builds, security, migration, focused Go matrix and Linux race evidence passed.

## Next gate

PD-D8-S02 — Platform authority integration certification is the next ready cross-repository slice. It must compare semantic contracts, prove sole shared-authority writing and local projection reads, inventory compatibility usage, and rehearse bounded retirement/cutover/rollback before any production activation. This handoff does not authorize a production cutover.

WSL Docker evidence: docker buildx build --platform linux/amd64 and linux/arm64 both completed successfully from C:\MyGitProjects\tockrPlatform on the exact Platform candidate. The Windows full/local wrapper's Docker children are TOOL_FAIL because no Windows docker executable is installed; the equivalent WSL commands are the authoritative build evidence.

