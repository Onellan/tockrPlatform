# Authorization, Privacy and Security Acceptance Validation

Load for `AUTH` surfaces: authentication, authorization, privacy, sensitive fields or security-relevant input.

Test the acceptance boundary, not implementation style:

- exercise allowed and denied behavior at the server-side boundary;
- cover each material role/workspace/project relationship required by authority;
- inspect affected HTML, API, report, export and indirect projections that can expose or mutate protected data;
- search response/output bodies for sensitive fields that must be absent;
- verify direct crafted requests, not only visible/hidden UI controls;
- use repository security scanners only when the changed surface makes them relevant and tooling is already available;
- never expose secrets or generated credentials in evidence/output.

Do not repeat the engineering reviewer's design/style review. Inspect implementation paths only to locate the acceptance-relevant exposure/mutation seams and design executable evidence.