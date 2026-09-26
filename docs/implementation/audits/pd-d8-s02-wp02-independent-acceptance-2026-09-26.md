# PD-D8-S02 WP-PD8S02-02 tester acceptance — 2026-09-26

## Exact source code candidates

- Platform `3115caad66b4086535720a03ccf08c2766666edd`
- CTRL `4710090b2d6019bc1686945828bc1e90eb7f6a8d`
- IMS `69344937bcb8e030976dd2c67447d7d2d41861a8`

Evidence publication commits: Platform `19d4193524b1b2fe551b31059af244cb3b057329`,
CTRL `55206a9f5092e00b71d2f0b9ffea4d9e18b26b32`, IMS
`a839a211d48cc04b462f7e1916b1ccd35867c0f3`.

## Acceptance matrix

| AC | Result | Evidence |
| --- | --- | --- |
| AC-D8S02-02A Platform writer inventory is complete | PASS | Platform identity, Organisation, Workspace, product, access, membership-command and outbox seams listed |
| AC-D8S02-02B CTRL normal requests use local projections | PASS | CTRL server/SQLite projection paths and focused HTTP/SQLite tests |
| AC-D8S02-02C IMS normal requests use local projections | PASS | IMS server/SQLite projection paths and focused HTTP/SQLite tests |
| AC-D8S02-02D ordinary requests have no synchronous Platform dependency | PASS | disabled production lookup seam; only explicit session and membership-command calls are Platform-bound |
| AC-D8S02-02E product-owned project, discipline, programme, control and commercial data remains local | PASS | consumer writer inventory and projection boundary inspection |
| AC-D8S02-02F no dual shared-authority writer is enabled | PASS | mutually exclusive write-mode validation and request routing |
| AC-D8S02-02G exact-candidate focused and repository-wide Go suites pass | PASS | Platform, CTRL and IMS command outputs recorded in inventory |

## Tester verdict

**PASS — WP-PD8S02-02 is accepted and terminal.** Platform is the only
shared-authority writer in Platform-authoritative mode. The pre-cutover local
compatibility mode remains a separately selected rollback stage; no production
configuration was changed by this evidence slice.

