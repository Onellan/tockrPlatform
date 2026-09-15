# UI Validation

Read when risk includes `UI` or a user-visible interaction materially changes.

- use disposable local data and a safe local port;
- verify the changed workflow at the real browser/runtime boundary when lower seams cannot prove it;
- cover relevant desktop/mobile layouts and applicable validation, denied actions, empty/unknown/zero states, and recovery from server-side errors;
- preserve accessible form semantics and server-side fallback behavior where applicable;
- verify security/authorization separately at the server boundary; visible/hidden controls are not permission evidence.

Limit browser coverage to changed and materially neighboring workflows.