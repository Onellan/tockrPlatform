#!/usr/bin/env python3
"""Append interruption-safe delivery telemetry for reasoning routing."""
from __future__ import annotations

import argparse
from contextlib import contextmanager
import json
import os
import re
import subprocess
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from result_contract import current_candidate
from shared_delivery_contract import ALLOWED_RISK_CODES

ROOT = Path(__file__).resolve().parents[1]
PHASES = {"plan", "implement", "repair-implement", "review", "test", "release-gate", "completion-gate"}
IMPLEMENTATION_PHASES = {"implement", "repair-implement"}
EVENTS = {"started", "completed"}
RESULTS = {"Pass", "Fail", "Blocked", "Paused", "Interrupted", "Reroute", "Success", "Failure"}
KINDS = {"routine", "defect", "migration", "authorization", "ui", "other"}
RISK_LEVELS = {"R", "E", "H"}
RISK_CODES = ALLOWED_RISK_CODES


def slug(value: str) -> str:
    result = re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-")
    if not result:
        raise ValueError("item must contain at least one alphanumeric character")
    return result


def codes(value: str) -> list[str]:
    result = sorted({part.strip().upper() for part in value.split(",") if part.strip()})
    unknown = sorted(set(result) - RISK_CODES)
    if unknown:
        raise ValueError("unknown risk code(s): " + ", ".join(unknown))
    return result


def nullable_int(value: int) -> int | None:
    return None if value < 0 else value


def nullable_float(value: float) -> float | None:
    return None if value < 0 else value


def current_git(*args: str) -> str | None:
    try:
        result = subprocess.run(["git", *args], cwd=ROOT, check=True, capture_output=True, text=True)
    except (OSError, subprocess.CalledProcessError):
        return None
    value = result.stdout.strip()
    return value or None


def existing_records(path: Path, identity: tuple[str, str, int, str | None, str | None, str]) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    if not path.exists():
        return records
    try:
        lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
    except OSError:
        return records
    for line in lines:
        if not line.strip():
            continue
        try:
            record = json.loads(line)
        except (json.JSONDecodeError, UnicodeError):
            # A torn append is not evidence and cannot block later valid
            # invocation records. Preserve the original line on disk.
            continue
        if not isinstance(record, dict):
            continue
        work = record.get("work") or {}
        current = (
            record.get("phase"), record.get("agent"), record.get("attempt"),
            work.get("package"), work.get("slice"), record.get("invocation_id"),
        )
        if current == identity:
            records.append(record)
    return records


def _candidate_identity(candidate: dict[str, Any] | None) -> tuple[Any, Any, Any]:
    value = candidate or {}
    return value.get("commit"), value.get("fingerprint"), value.get("id")


def _invocation_metadata(record: dict[str, Any]) -> tuple[Any, ...]:
    work = record.get("work") or {}
    return (
        record.get("item"),
        record.get("phase"),
        record.get("agent"),
        record.get("attempt"),
        record.get("invocation_id"),
        record.get("repair_cycle"),
        work.get("package"),
        work.get("slice"),
        work.get("kind"),
        work.get("scope"),
        work.get("risk_level"),
        tuple(work.get("risk_codes") or []),
        record.get("reasoning_effort"),
        _candidate_identity(record.get("candidate")),
    )


@contextmanager
def append_lock(path: Path):
    """Serialize read/check/append across concurrent recorder processes."""
    lock_path = path.with_name(f".{path.name}.lock")
    lock_path.parent.mkdir(parents=True, exist_ok=True)
    with lock_path.open("a+b") as lock:
        if os.name == "nt":
            import msvcrt
            lock.seek(0, os.SEEK_END)
            if lock.tell() == 0:
                lock.write(b"0")
                lock.flush()
            lock.seek(0)
            msvcrt.locking(lock.fileno(), msvcrt.LK_LOCK, 1)
            try:
                yield
            finally:
                lock.seek(0)
                msvcrt.locking(lock.fileno(), msvcrt.LK_UNLCK, 1)
        else:
            import fcntl
            fcntl.flock(lock.fileno(), fcntl.LOCK_EX)
            try:
                yield
            finally:
                fcntl.flock(lock.fileno(), fcntl.LOCK_UN)


def _append_record(path: Path, identity: tuple[str, str, int, str | None, str | None, str], args: argparse.Namespace, record_value: dict[str, Any]) -> None:
    with append_lock(path):
        records = existing_records(path, identity)
        if any(record.get("event") == "started" for record in records) or any(record.get("event") == "completed" for record in records):
            if args.event == "started":
                raise ValueError("delivery invocation already has a start or completion; increment attempt for a new invocation")
            if any(record.get("event") == "completed" for record in records):
                raise ValueError("delivery invocation already has a completion")
        if args.event == "completed" and not any(record.get("event") == "started" for record in records):
            raise ValueError("delivery completion requires a matching started event")
        if args.event == "completed":
            started = next(record for record in records if record.get("event") == "started")
            if _invocation_metadata(started) != _invocation_metadata(record_value):
                raise ValueError("delivery completion metadata and candidate do not match the started invocation")
        with path.open("a", encoding="utf-8", newline="\n") as stream:
            stream.write(json.dumps(record_value, separators=(",", ":")) + "\n")


def record(args: argparse.Namespace) -> Path:
    if args.phase not in PHASES:
        raise ValueError(f"unknown phase: {args.phase}")
    if args.event not in EVENTS:
        raise ValueError(f"unknown event: {args.event}")
    if args.attempt < 1:
        raise ValueError("attempt must be positive")
    package = args.work_package or None
    work_slice = args.work_slice or None
    invocation_id = args.invocation_id.strip()
    if not invocation_id:
        raise ValueError("invocation_id must not be empty")
    kind = args.work_kind or None
    risk_level = args.risk_level or None
    risk_codes = codes(args.risk_codes)
    if work_slice and not package:
        raise ValueError("work slice requires a work package")
    if kind and not package:
        raise ValueError("work kind requires a work package")
    if (package or work_slice or kind) and args.phase not in IMPLEMENTATION_PHASES:
        raise ValueError("work-package/work-kind telemetry is valid only for implementation phases")
    if kind and kind not in KINDS:
        raise ValueError(f"unknown work kind: {kind}")
    if risk_level and risk_level not in RISK_LEVELS:
        raise ValueError(f"unknown risk level: {risk_level}")
    if risk_codes and not risk_level:
        raise ValueError("risk codes require an explicit risk level")
    if args.event == "started" and args.result:
        raise ValueError("started telemetry cannot have a terminal result")
    if args.event == "completed" and not args.result:
        raise ValueError("completed telemetry requires a result")
    if args.result and args.result not in RESULTS:
        raise ValueError(f"unknown result: {args.result}")

    report_root = args.report_root or ROOT / ".codex" / "agent-evals"
    report_root.mkdir(parents=True, exist_ok=True)
    path = report_root / f"delivery-{slug(args.item)}.jsonl"
    identity = (args.phase, args.agent, args.attempt, package, work_slice, invocation_id)

    input_tokens = nullable_int(args.input_tokens) if args.event == "completed" else None
    output_tokens = nullable_int(args.output_tokens) if args.event == "completed" else None
    total_tokens = input_tokens + output_tokens if input_tokens is not None and output_tokens is not None else None
    record_value = {
        "schema_version": 3,
        "event": args.event,
        "recorded_at_utc": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "item": args.item,
        "phase": args.phase,
        "agent": args.agent,
        "attempt": args.attempt,
        "invocation_id": invocation_id,
        "repair_cycle": None if args.repair_cycle < 0 else args.repair_cycle,
        "result": args.result if args.event == "completed" else None,
        "reasoning_effort": args.reasoning_effort if args.reasoning_effort != "unavailable" else None,
        "metric_source": args.metric_source if args.event == "completed" else None,
        "work": {
            "package": package,
            "slice": work_slice,
            "kind": kind,
            "scope": "slice" if work_slice else "work-package" if package else "phase",
            "risk_level": risk_level,
            "risk_codes": risk_codes,
        },
        "candidate": current_candidate(ROOT),
        "repository": {"head": current_git("rev-parse", "HEAD"), "branch": current_git("branch", "--show-current")},
        "metrics": {
            "turns": nullable_int(args.turns) if args.event == "completed" else None,
            "tool_calls": nullable_int(args.tool_calls) if args.event == "completed" else None,
            "input_tokens": input_tokens,
            "cached_input_tokens": nullable_int(args.cached_input_tokens) if args.event == "completed" else None,
            "output_tokens": output_tokens,
            "total_tokens": total_tokens,
            "elapsed_seconds": nullable_float(args.elapsed_seconds) if args.event == "completed" else None,
        },
    }
    _append_record(path, identity, args, record_value)
    return path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--item", required=True)
    parser.add_argument("--phase", required=True)
    parser.add_argument("--agent", required=True)
    parser.add_argument("--attempt", type=int, required=True)
    parser.add_argument("--invocation-id", required=True)
    parser.add_argument("--event", choices=sorted(EVENTS), default="completed")
    parser.add_argument("--result")
    parser.add_argument("--reasoning-effort", choices=("high", "medium", "unavailable"), default="unavailable")
    parser.add_argument("--metric-source", default="unavailable")
    parser.add_argument("--work-package")
    parser.add_argument("--work-slice")
    parser.add_argument("--work-kind")
    parser.add_argument("--risk-level")
    parser.add_argument("--risk-codes", default="")
    parser.add_argument("--repair-cycle", type=int, default=-1)
    parser.add_argument("--turns", type=int, default=-1)
    parser.add_argument("--tool-calls", type=int, default=-1)
    parser.add_argument("--input-tokens", type=int, default=-1)
    parser.add_argument("--cached-input-tokens", type=int, default=-1)
    parser.add_argument("--output-tokens", type=int, default=-1)
    parser.add_argument("--elapsed-seconds", type=float, default=-1)
    parser.add_argument("--report-root", type=Path)
    args = parser.parse_args()
    try:
        print(record(args))
    except ValueError as error:
        parser.error(str(error))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
