---
name: codebase-design
description: Design and review Tockr Platform modules, interfaces, seams and adapters for depth, locality, leverage and testability.
---

# Tockr Platform Codebase Design

Design **deep modules**: small stable interfaces that hide meaningful complexity and keep governance knowledge local.

## Rules

1. Prefer extending an existing module over creating a conventional new layer.
2. Put invariants where every material writer crosses them.
3. Keep HTTP parsing/auth/validation/response shaping at the request boundary; place shared invariants behind a common seam when multiple writers need them.
4. Keep SQLite representation local to `internal/db/sqlite`.
5. Keep email, assertion and future provider-specific details behind explicit adapter seams.
6. Do not create interfaces for hypothetical substitution; create an interface only where a real Platform provider, consumer or test seam varies.
7. Respect current single-instance/SQLite architecture.
8. Preserve canonical product meaning over internal naming convenience.

## Depth check

For each material abstraction ask: callers, required caller knowledge, hidden complexity, smaller possible interface, caller leverage and behavioural test seam.

## Deletion test

If deleting the abstraction makes complexity disappear, it is probably pass-through. If rules/knowledge would scatter into several callers, it is likely useful.

## Review signals

Look closely at repeated handler rules, forwarding-only types, many abstractions for one behaviour, persistence details leaking upward, provider details leaking sideways, or one domain rule requiring scattered edits.
