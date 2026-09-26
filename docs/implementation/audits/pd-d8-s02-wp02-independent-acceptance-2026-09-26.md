# PD-D8-S02 WP-PD8S02-02 tester acceptance — 2026-09-26

## Exact source code candidates

- Platform `3115caad66b4086535720a03ccf08c2766666edd`
- CTRL `677dd0dc5573b0bb7ac6c42d2d20c907505e6683`
- IMS `0a8750fbac24ff6ceca3b8fd13afd70b503f2e92`

Evidence publication commits: Platform `8e6593200ba624566407b29c58448f4f9a68d139`,
CTRL `be821c5c68097f36aac07104884ec05fa66b7d13`, IMS
`817f3f0412a253b9eb0d4eed0f35e3ded3379736`.

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

