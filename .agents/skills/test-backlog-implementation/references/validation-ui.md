# User-Visible Acceptance Validation

Load for `UI` only when authoritative acceptance requires user-visible interaction/rendering/accessibility-sensitive workflow evidence, or when lower seams cannot prove the condition completely.

Browser capability is **lazy**. A changed route/template/JavaScript file does not by itself require browser startup. If HTTP/store/template evidence fully proves the authoritative condition and there is no visual/interaction acceptance requirement, record that lower-seam evidence and do not launch the application/browser.

When browser evidence is required:

- use disposable local data and non-production credentials on a safe local port;
- prepare one runtime/session and reuse it across compatible `AC##` rows where safe;
- verify the complete affected workflow at relevant desktop and narrow/mobile widths;
- cover applicable validation errors, denied actions, empty/unknown/zero states, stale/recovery behavior, authoritative labels, accessibility-sensitive interaction, and final outcomes.

Verify server-side authorization/security separately through the security reference; UI hiding is never security evidence.

Do not expose generated credentials in the report. Store temporary browser output outside the repository where practical and remove only processes/artifacts created by this test run.