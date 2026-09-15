---
name: architecture-health-review
description: Perform an explicit read-only architecture-health review for evidence-backed depth, locality and boundary problems.
---

# tockrPlatform Architecture Health Review

This skill is opt-in and read-only. It is not part of routine feature delivery.

Review current code plus architecture/standards for evidence-backed problems:

- shallow/pass-through modules;
- duplicated invariants;
- shotgun change hotspots;
- domain meaning scattered across packages;
- SQLite/provider details leaking through seams;
- auth/privacy rules duplicated or missing;
- historical truth represented weakly;
- assertion/event/email boundary at risk;
- unbounded resource/query behaviour;
- side effects without recoverable failure semantics.

Use depth/locality/deletion tests. Rank findings by observed impact/churn/risk and propose bounded remediation opportunities; do not refactor automatically or create speculative architecture work merely because an alternative pattern exists.
