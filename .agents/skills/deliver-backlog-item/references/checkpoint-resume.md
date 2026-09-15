# Checkpoint and Resume Protocol

Maintain one ignored checkpoint at `.codex/delivery-state/<item>.md` for every active delivery. It is resumable delivery evidence, never product authority or completion documentation.

## Compact required record

Persist IDs/hashes/results, not repeated specialist prose:

```text
item=<number/title>
plan=<path>; plan_sha=<sha256>
authority_hashes=<path:sha,...>
head=<sha>; branch=<branch>
status=<git status fingerprint/full baseline ref>
user_owned=<paths>
delivery_status=<active|paused|blocked|release-ready|complete>
pause_reason=<none|codex_usage_limit|awaiting_publish_authority|awaiting_ci|other>
last_completed_phase=<phase>
next_phase=<plan|implement|review|test|repair-loop|release-gate|completion-gate|complete>
metrics_path=.codex/agent-evals/delivery-<item-slug>.jsonl
metric_attempts=<invocation-key:n,...|none>
metric_inflight=<phase:agent:work-package:slice:attempt|none>
package_progress=<WP1:pending|active|complete,...|none>
package_routing=<WP1:kind:risk[codes]:recommendation:authorized:runtime:gate-or-none,...|none>

PlanState=<compact record|none>
ImplementState=<compact record|none>
ReviewState=<compact record|none>
TestState=<compact record|none>
ReleaseState=<compact record|none>

repair_iteration=<n|none>
repair_source=<reviewer|tester|release-ci|none>
repair_stage=<implement|review|test|none>
open_findings=<fingerprints|none>
addressed_findings=<fingerprints|none>
updated=<timestamp>
```

`metrics_path` points to the ignored delivery telemetry stream; do not copy token/tool/runtime values into the checkpoint. The JSONL stream is the durable source of telemetry invocation identity.

`metric_attempts` stores the highest **started** invocation number for each telemetry identity. Identity is `phase + agent + optional planned work package + optional slice`; use `-` for absent work-package/slice fields in compact checkpoint text. `metric_inflight` is a hint for the invocation whose `started` event exists but whose `completed` event has not yet been observed. Reconcile both fields against the JSONL stream on resume; the stream wins if the checkpoint was not updated before a hard interruption.

`package_progress` is used only when implementation is package-routed. It prevents quota/session resume from re-running already completed packages. `package_routing` stores compact recommendation/authorization metadata for not-yet-started and completed packages; it is agent-control evidence, not product authority. Never copy evaluation reports or telemetry payloads into the checkpoint.

Work-package/slice identity is telemetry attribution only. It does not create product authority or extra acceptance gates. A slice must belong to a planned work package. Do not split an otherwise cohesive implementation solely to improve telemetry granularity. Package-level invocation is legitimate when approved reasoning routing requires a distinct runtime for that package.

`TestState` must preserve fingerprint, tested HEAD/diff identity, exact plan/authority fingerprint inputs, verdict, every authoritative AC as a compact state/evidence mapping, and finding references. Settled passing rows remain as forms such as `AC01:P:E01,E02`; do not restore their requirement/subcase prose merely for checkpointing. Non-pass rows/findings preserve the diagnostic detail required to resume safely.

`ReleaseState` must preserve exact candidate SHA, candidate equivalence, CI status/run and failed job/step when relevant.

For all phase states, successful command/observation evidence is reference-only: retain `E##`, check/seam identity and `PASS` plus a useful small metric if needed. Do not persist successful stdout, package/setup/cache/scanner logs, browser noise or repeated success prose. `Fail`, `Blocked`, mixed, flaky or unexpected evidence keeps the smallest reproducible diagnostic.

Detailed prose belongs only with an open finding/blocker where reproduction would otherwise be lost. Never record secrets, quota percentages or reset times.

## Invocation boundary invariant

For every specialist/gate invocation use this order:

```text
choose next attempt for exact phase/agent/work-unit identity
-> append telemetry event=started
-> persist metric_attempts + metric_inflight without advancing product phase/package
-> invoke specialist/gate
-> append telemetry event=completed when an outcome returns
-> clear metric_inflight + persist phase/package/gate checkpoint
-> start next action
```

The `started` event must be written **before** delegation so a 5-hour/weekly usage-window cutoff or process/session interruption cannot make an already-started invocation disappear from telemetry.

If the execution surface returns an interrupted/quota outcome and exact counters, complete the same attempt with `Result=Paused` (or `Interrupted` for a non-quota interruption) plus those exact counters. If the process stops before any outcome/counters return, leave the `started` event orphaned. Do not fabricate a completion merely to make the stream tidy.

A medium implementer that detects a pre-mutation routing mismatch closes its invocation as `Reroute` when an outcome is available. Preserve exact counters if exposed, record the compact escalation reason, and invoke the same package using the high runtime. `Reroute` is agent-control evidence, not product failure or a repair cycle.

Telemetry recording is observability, not a quality gate. If the execution surface exposes no exact token/tool/runtime counter, leave it unavailable; never estimate. A telemetry-script problem is reported separately and must not turn a valid product phase into `Fail` or `Blocked`.

Quota exhaustion sets `paused/codex_usage_limit`, never product `Fail/Blocked`, and stales no unchanged evidence.

## Lifecycle

```text
new                         -> next=plan
plan accepted/routed        -> last=plan; next=implement
whole-plan implementation   -> last=implement; next=review
package complete            -> remain next=implement until all packages complete
all packages complete       -> last=implement; next=review
review Pass                 -> last=review; next=test
tester Pass                 -> last=test; next=release-gate
release CI green            -> last=release-gate; next=completion-gate
terminal truth done         -> last=completion-gate; next=complete
```

Failed review/test/CI records only current finding fingerprints + reproduction/evidence refs before entering repair.

## Reasoning-routing resume

For every not-yet-started package, the planner signature (`kind`, `risk[codes]`) is the pre-work input to `python scripts/recommend_implementer_reasoning.py`. The legacy PowerShell command is a thin compatibility wrapper.

On resume:

1. preserve actual runtime/evidence for completed packages;
2. do not rerun a completed package because the routing policy changed;
3. re-run the recommender for each **not-yet-started** package so a newly approved policy may affect future work only;
4. if a prior recommendation was `medium` but authorization remains absent/stale, continue high without blocking delivery;
5. if a package was in-flight at quota cutoff, reconcile telemetry/package state before deciding whether a new invocation is needed;
6. a routing-policy/agent-config-only delta does not stale prior product review/test evidence by itself.

If all remaining packages are authorized high and package-routed mode has **not** started, delivery may preserve whole-plan mode. Once package-routed implementation has started, remain package-routed for the remaining packages so checkpoint identity and already-completed package evidence stay coherent.

## Resume validation

On continuation recompute plan/authority hashes, HEAD/branch and repository status/fingerprint; protect all recorded user-owned paths. Then reconcile telemetry before invoking anything:

1. read the JSONL stream and resolve identity as `phase + agent + work package + slice`;
2. find the maximum started attempt for each exact identity;
3. find any `started` event without a matching `completed` event for that identity;
4. treat each orphan as an already-consumed/incomplete invocation with unavailable counters unless exact counters can be recovered from the execution surface;
5. never reuse an orphaned attempt number;
6. the next real invocation uses `max(started attempt) + 1` for that exact identity;
7. update `metric_attempts`/`metric_inflight` to match the stream without changing product/review/test evidence;
8. reconcile `package_progress` against saved package-scoped implementation states; telemetry alone does not prove a package completed.

This permits `implement/backlog_implementer/WP1/-/attempt=1` and `implement/backlog_implementer_medium/WP2/-/attempt=1` to coexist: attempts are monotonic **within an exact runtime/work unit**, not globally across all packages. Phase totals still aggregate both invocations.

An orphaned start is not itself a product failure and does not force re-execution of a phase/package that the normal delivery checkpoint already proves complete. Conversely, telemetry does not prove work completed; normal phase/package-state evidence still controls resume.

- no checkpoint/fresh delivery -> establish baseline, start `plan`;
- plan/substantive authority meaning changed -> earliest affected phase;
- repository changed -> classify with `repair-resume.md` before invalidating evidence;
- valid package-routed implementation -> resume first non-complete ordered package;
- valid `repair-loop` -> resume exact `repair_stage`;
- valid `release-gate` -> continue tester-state/candidate/CI verification, not whole delivery;
- valid `completion-gate` -> require recorded exact-candidate green CI and classify only later changes;
- valid `complete` -> report while terminal/current-state evidence still matches.

Graded invalidation:

- product/runtime change -> reviewer + tester evidence stale;
- evidence-harness-only -> targeted independent tester revalidation;
- completion/docs-only -> DOC validation when acceptance meaning unchanged;
- publication-only -> preserve tester Pass only with positive equivalence;
- reasoning-policy/agent-control-only -> does not stale completed product evidence;
- quota/process interruption -> stales nothing.

When a narrow category cannot be positively established, use the safer broader invalidation.

## Idempotency

Never rerun a completed unchanged phase/package merely because execution paused. A completed repair implementation at `repair_stage: review` is not rerun; a passed repair review at `repair_stage: test` is not rerun; a saved tester Pass with matching fingerprint is reusable; a green CI result is valid only for its exact candidate SHA.

Telemetry idempotency is event-based:

- one `started` event maximum per phase + agent + work package + slice + attempt;
- one `completed` event maximum for that exact identity;
- a `completed` event closes the same attempt and never creates a second invocation;
- different work packages/slices/runtime agents may use the same numeric attempt because their identities differ;
- an orphaned `started` event remains part of delivery cost coverage and is never deleted/reused to improve metrics.

Reasoning-routing idempotency is package-state based:

- recommendations may be recomputed for pending packages;
- authorization changes apply only before a package starts;
- a completed package retains the reasoning/runtime that actually executed it;
- `Reroute` may add a high-runtime invocation for the same package but does not erase the medium attempt.

## Terminal override

If the item becomes `Implemented`, verify terminal metadata is backed by the required exact-candidate release evidence. Valid terminal state overrides the checkpoint. Terminal metadata without required release evidence is inconsistent completion truth, not permission to silently accept or reopen the item.

## Economy

Checkpoint only durable phase/package/gate boundaries, repair substages, compact package routing choices, and the telemetry in-flight reservation needed for interruption safety. Store compact phase records and references to evidence; do not copy agent narratives, successful command logs, product requirements, plan prose, evaluation reports or telemetry payloads into the checkpoint.
