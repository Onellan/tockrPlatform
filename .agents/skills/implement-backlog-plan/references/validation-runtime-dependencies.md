# Runtime and Dependency Validation

Read when risk includes `OPS`, `DEPLOY` or `DEP`, or when imports/build graph/external/runtime behavior materially changes.

## Runtime and side effects

Where applicable:

- verify timeout/cancellation behavior at external or long-running boundaries;
- verify retry/idempotency/duplicate-delivery semantics for repeatable side effects;
- verify partial failure and restart/recovery when persisted work spans process lifetime;
- verify errors are diagnosable without leaking secrets;
- run startup/health, storage, container or deployment smoke checks at the real runtime boundary.

## Dependencies and build graph

When imports, `go.mod`, `go.sum`, build tags or dependency selection changed:

- run `go mod tidy -diff` or the repository-equivalent unexpected-diff check;
- verify affected build/platform compatibility, including `linux/arm64` where relevant;
- run `govulncheck`, licensing/security review or equivalent checks when supply-chain/runtime risk materially expands;
- confirm the dependency satisfies the repository's dependency-stewardship standard.

Do not update unrelated dependencies or add runtime machinery unrelated to the approved delta.