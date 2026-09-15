#!/usr/bin/env python3
"""Repository adapter for the shared, fail-closed implementer router."""
from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

from shared_delivery_contract import (
    ALLOWED_KINDS,
    ALLOWED_RISK_LEVELS,
    SCENARIOS,
    gate_supports,
    load_policy,
    normalize_codes,
    route_work,
)

ROOT = Path(__file__).resolve().parents[1]


def gate_records(report_root: Path) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    if not report_root.exists():
        return records
    for path in sorted(report_root.glob("agent-eval-gate-*.json"), key=lambda item: item.stat().st_mtime, reverse=True):
        try:
            value = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            continue
        if isinstance(value, dict):
            records.append({"path": str(path), "manifest": value})
    return records


def history(report_root: Path, effort: str, kind: str, risk: str, codes: list[str], minimum_samples: int) -> dict[str, Any]:
    totals: list[float] = []
    samples = 0
    for path in report_root.glob("delivery-*.jsonl") if report_root.exists() else []:
        try:
            lines = path.read_text(encoding="utf-8").splitlines()
        except OSError:
            continue
        for line in lines:
            try:
                record = json.loads(line)
            except json.JSONDecodeError:
                continue
            work = record.get("work") or {}
            try:
                record_codes = normalize_codes(work.get("risk_codes"))
            except ValueError:
                continue
            if (
                record.get("event") != "completed"
                or record.get("phase") not in {"implement", "repair-implement"}
                or record.get("reasoning_effort") != effort
                or work.get("risk_level") != risk
                or work.get("kind", kind) != kind
                or record_codes != codes
                or not work.get("package")
            ):
                continue
            samples += 1
            total = (record.get("metrics") or {}).get("total_tokens")
            if isinstance(total, (int, float)):
                totals.append(float(total))
    totals.sort()
    median = None if not totals else totals[len(totals) // 2] if len(totals) % 2 else (totals[len(totals) // 2 - 1] + totals[len(totals) // 2]) / 2
    return {
        "samples": samples,
        "exact_token_samples": len(totals),
        "median_total_tokens": round(median, 2) if median is not None else None,
        "enough_for_descriptive_comparison": samples >= minimum_samples,
    }


def parser() -> argparse.ArgumentParser:
    result = argparse.ArgumentParser(description=__doc__)
    result.add_argument("--work-package", required=True)
    result.add_argument("--plan", type=Path, required=True)
    result.add_argument("--work-kind", required=True, choices=sorted(ALLOWED_KINDS))
    result.add_argument("--risk-level", required=True, choices=sorted(ALLOWED_RISK_LEVELS))
    result.add_argument("--risk-codes", default="")
    result.add_argument("--policy-path", type=Path)
    result.add_argument("--report-root", type=Path)
    result.add_argument("--minimum-history-samples", type=int, default=3)
    return result


def main() -> int:
    args = parser().parse_args()
    if args.minimum_history_samples < 1:
        raise SystemExit("--minimum-history-samples must be positive")
    policy_path = args.policy_path or ROOT / ".codex" / "reasoning-routing-policy.json"
    report_root = args.report_root or ROOT / ".codex" / "agent-evals"
    from validate_plan_routing import declared_route, validate_plan

    plan_issues = validate_plan(args.plan, args.work_package)
    declared = declared_route(args.plan, args.work_package)
    try:
        supplied_codes = normalize_codes(args.risk_codes)
    except ValueError:
        supplied_codes = []
    policy, warnings = load_policy(policy_path)
    records = gate_records(report_root)
    scenario = SCENARIOS.get(args.work_kind)
    supporting = next((record for record in records if gate_supports(record["manifest"], scenario, report_root=report_root)), None) if scenario else None
    approved_gate_ids = {
        str(record["manifest"].get("run_id"))
        for record in records
        if record["manifest"].get("run_id") and gate_supports(record["manifest"], scenario, report_root=report_root)
    }
    decision = route_work(
        work_package=args.work_package,
        work_kind=args.work_kind,
        risk_level=args.risk_level,
        risk_codes=args.risk_codes,
        policy=policy,
        supporting_gate=supporting is not None,
        approved_gate_ids=approved_gate_ids,
        plan_route=declared,
        plan_issues=plan_issues,
        history={
            "comparable_threshold": args.minimum_history_samples,
            "high": history(report_root, "high", args.work_kind, args.risk_level, supplied_codes, args.minimum_history_samples),
            "medium": history(report_root, "medium", args.work_kind, args.risk_level, supplied_codes, args.minimum_history_samples),
            "note": "Delivery history is descriptive only; it never authorizes medium without paired quality evidence and an approved policy rule.",
        },
    )
    decision["policy_warnings"] = warnings
    decision["evidence_gate_run_id"] = supporting["manifest"].get("run_id") if supporting else None
    approved = next((rule for rule in policy.get("approved_rules", []) if rule.get("enabled") and rule.get("evidence_gate_run_id") in approved_gate_ids and rule.get("risk_level") == args.risk_level and args.work_kind in rule.get("work_kinds", []) and rule.get("risk_codes") == supplied_codes), None)
    decision["approved_rule_id"] = approved.get("id") if approved else None
    decision["approved_gate_run_id"] = approved.get("evidence_gate_run_id") if approved else None
    print(json.dumps(decision, indent=2))
    return 1 if decision["plan_routing_issues"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
