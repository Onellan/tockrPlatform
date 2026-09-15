# Security Validation

Read when risk includes `AUTH` or security-sensitive input/secrets materially change.

- prove allowed and denied behavior at the server boundary;
- inspect every affected HTML/API/report/export/projection path that can expose or mutate protected data;
- verify CSRF/request-safety and input-validation behavior where applicable;
- run repository security scanners/checks only when the changed surface warrants them or CI reproduction is required;
- keep secrets out of logs, diagnostics and test output.

A hidden UI control is never sufficient authorization evidence.