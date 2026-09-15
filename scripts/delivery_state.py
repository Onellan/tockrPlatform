#!/usr/bin/env python3
"""Compact, resumable backlog delivery state contract."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import tempfile
from pathlib import Path
from typing import Any

from result_contract import current_candidate
from shared_delivery_contract import ALLOWED_RISK_CODES
from validate_plan_routing import declared_packages, declared_route, validate_plan

SCHEMA_VERSION = 1
ROOT = Path(__file__).resolve().parents[1]
PHASES = {"plan", "implement", "repair-implement", "review", "test", "release-gate", "completion-gate"}
PACKAGE_STATUSES = {"pending", "running", "pass", "fail", "blocked"}
REVIEW_STATUSES = {"pending", "pass", "fail", "blocked"}
WORK_KINDS = {"routine", "defect", "migration", "authorization", "ui", "other"}
RISK_LEVELS = {"R", "E", "H"}
RISK_CODES = ALLOWED_RISK_CODES
MATERIAL_RISK_CODES = RISK_CODES - {"UI"}


def valid_candidate(value: object) -> bool:
    return isinstance(value, dict) and all(
        isinstance(value.get(key), str) and value[key].strip()
        for key in ("commit", "fingerprint", "id")
    )


def _candidate_identity(value: dict[str, Any]) -> tuple[Any, Any, Any]:
    return value.get("commit"), value.get("fingerprint"), value.get("id")


def _relative_path(path: str, root: Path) -> str:
    candidate = Path(path)
    if candidate.is_absolute():
        try:
            candidate = candidate.resolve().relative_to(root.resolve())
        except ValueError as error:
            raise ValueError(f"authority path is outside the repository: {path}") from error
    normalized = candidate.as_posix()
    if not normalized or normalized == "." or normalized.startswith("../") or "/../" in normalized:
        raise ValueError(f"authority path is invalid: {path}")
    return normalized


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    try:
        with path.open("rb") as stream:
            for chunk in iter(lambda: stream.read(1024 * 1024), b""):
                digest.update(chunk)
    except OSError as error:
        raise ValueError(f"authority path is not readable: {path}") from error
    return digest.hexdigest()


def _authority_binding(plan: str, root: Path, authority_paths: list[str] | None) -> dict[str, Any]:
    paths = [plan, *(authority_paths or [])]
    hashes: dict[str, str] = {}
    for path in paths:
        relative = _relative_path(path, root)
        if relative in hashes:
            continue
        resolved = root / relative
        if not resolved.is_file():
            raise ValueError(f"authority path does not exist: {relative}")
        hashes[relative] = _sha256(resolved)
    return {"plan": _relative_path(plan, root), "hashes": hashes}


def new_state(item: str, plan: str, candidate: dict[str, Any], *, root: Path = ROOT,
              authority_paths: list[str] | None = None) -> dict[str, Any]:
    if not valid_candidate(candidate):
        raise ValueError("delivery state candidate must contain commit, fingerprint and id")
    authority = _authority_binding(plan, root, authority_paths)
    return {
        "schema_version": SCHEMA_VERSION,
        "item": item,
        "plan": authority["plan"],
        "candidate": dict(candidate),
        "authority": authority,
        "next": {"phase": "plan", "package": None, "stage": "start"},
        "packages": [],
        "routing": [],
        "evidence_ids": [],
        "finding_ids": [],
        "review": {"status": "pending"},
        "test": {"status": "pending"},
        "release": {"status": "pending"},
        "metric_attempts": [],
        "metric_inflight": None,
    }


def validate_state(state: object, *, root: Path = ROOT) -> list[str]:
    issues: list[str] = []
    if not isinstance(state, dict):
        return ["delivery state must be an object"]
    if state.get("schema_version") != SCHEMA_VERSION:
        issues.append("delivery state schema version is invalid")
    if not isinstance(state.get("item"), str) or not state["item"].strip():
        issues.append("delivery state item is required")
    if not isinstance(state.get("plan"), str) or not state["plan"].strip():
        issues.append("delivery state plan is required")
    candidate = state.get("candidate")
    if not valid_candidate(candidate):
        issues.append("delivery state candidate must contain commit, fingerprint and id")
    packages = state.get("packages")
    plan_path: Path | None = None
    authority = state.get("authority")
    if not isinstance(authority, dict) or not isinstance(authority.get("plan"), str) or not authority["plan"].strip():
        issues.append("delivery state authority plan is required")
    else:
        if state.get("plan") != authority["plan"]:
            issues.append("delivery state plan does not match authority plan")
        hashes = authority.get("hashes")
        if not isinstance(hashes, dict) or not hashes:
            issues.append("delivery state authority hashes are required")
        else:
            if authority["plan"] not in hashes:
                issues.append("delivery state authority hashes must include the plan")
            for path, digest in hashes.items():
                if not isinstance(path, str) or not path.strip() or not isinstance(digest, str) or len(digest) != 64 or any(char not in "0123456789abcdef" for char in digest):
                    issues.append(f"delivery state authority hash is invalid: {path}")
    if isinstance(authority, dict) and isinstance(authority.get("plan"), str) and authority["plan"].strip():
        try:
            plan_path = root / _relative_path(authority["plan"], root)
        except ValueError:
            plan_path = None
        if plan_path is None or not plan_path.is_file():
            issues.append("delivery state plan is unavailable for route validation")
        else:
            plan_issues = validate_plan(plan_path)
            issues.extend(f"delivery plan routing: {issue}" for issue in plan_issues)
            declared = set(declared_packages(plan_path))
            if isinstance(packages, list):
                for package in packages:
                    if isinstance(package, dict) and isinstance(package.get("id"), str) and package["id"] not in declared:
                        issues.append(f"delivery package is not declared by the plan: {package['id']}")
    for field in ("routing", "packages", "evidence_ids", "finding_ids", "metric_attempts"):
        if not isinstance(state.get(field), list):
            issues.append(f"delivery state {field} must be a list")
        elif field in {"evidence_ids", "finding_ids"} and any(not isinstance(value, str) or not value.strip() for value in state[field]):
            issues.append(f"delivery state {field} must contain non-empty identifiers")
    for field in ("review", "test", "release"):
        value = state.get(field)
        if not isinstance(value, dict) or value.get("status") not in REVIEW_STATUSES:
            issues.append(f"delivery state {field} must contain a valid status")
    pointer = state.get("next")
    if not isinstance(pointer, dict) or pointer.get("phase") not in PHASES or not pointer.get("stage"):
        issues.append("delivery state next pointer must contain a valid phase and stage")
    seen: set[str] = set()
    if isinstance(packages, list):
        for package in packages:
            if not isinstance(package, dict) or not isinstance(package.get("id"), str) or not package["id"].strip():
                issues.append("each delivery package needs an id")
                continue
            if package["id"] in seen:
                issues.append(f"duplicate delivery package: {package['id']}")
            seen.add(package["id"])
            if package.get("status") not in PACKAGE_STATUSES:
                issues.append(f"invalid status for delivery package: {package['id']}")
    if isinstance(pointer, dict) and pointer.get("package") is not None:
        package_id = pointer.get("package")
        if package_id not in seen:
            issues.append(f"next pointer references an unknown package: {package_id}")
        elif any(package.get("id") == package_id and package.get("status") == "pass" for package in packages if isinstance(package, dict)):
            issues.append(f"next pointer would replay completed package: {package_id}")
    routing = state.get("routing")
    routed_packages: set[str] = set()
    if isinstance(routing, list):
        for decision in routing:
            if not isinstance(decision, dict) or any(not isinstance(decision.get(key), str) or not decision[key].strip() for key in ("package", "kind", "risk_level", "authorized_effort", "runtime_agent")):
                issues.append("each routing decision needs package, kind, risk_level, authorized_effort and runtime_agent")
                continue
            package_id = decision["package"]
            if package_id in routed_packages:
                issues.append(f"duplicate routing decision: {package_id}")
            routed_packages.add(package_id)
            if plan_path is not None and plan_path.is_file():
                route = declared_route(plan_path, package_id)
                if route is None:
                    issues.append(f"routing decision has no unique plan Route signature: {package_id}")
                elif decision.get("kind") != route["kind"] or decision.get("risk_level") != route["risk"] or decision.get("risk_codes") != route["risk_codes"]:
                    issues.append(f"routing decision does not match the plan Route signature: {package_id}")
            if decision["kind"] not in WORK_KINDS:
                issues.append(f"invalid routing work kind: {package_id}")
            if decision["risk_level"] not in RISK_LEVELS:
                issues.append(f"invalid routing risk level: {package_id}")
            risk_codes = decision.get("risk_codes")
            if not isinstance(risk_codes, list) or any(not isinstance(code, str) or code not in RISK_CODES for code in risk_codes):
                issues.append(f"invalid routing risk_codes for {package_id}")
            if package_id not in seen:
                issues.append(f"routing decision references an unknown package: {package_id}")
            medium_eligible = decision["kind"] == "routine" and decision["risk_level"] == "R" and isinstance(risk_codes, list) and not set(risk_codes) & MATERIAL_RISK_CODES
            expected_effort = decision["authorized_effort"] if medium_eligible else "high"
            if decision["authorized_effort"] not in {"high", "medium"} or decision["authorized_effort"] != expected_effort:
                issues.append(f"routing effort is not risk-safe for {package_id}")
            expected_agent = "backlog_implementer_medium" if decision["authorized_effort"] == "medium" else "backlog_implementer"
            if decision["runtime_agent"] != expected_agent:
                issues.append(f"routing runtime does not match effort for {package_id}")
    if isinstance(pointer, dict) and pointer.get("phase") in {"implement", "repair-implement"} and pointer.get("package") is not None and pointer.get("package") not in routed_packages:
        issues.append(f"implementation pointer has no persisted routing decision: {pointer.get('package')}")
    inflight = state.get("metric_inflight")
    if inflight is not None:
        if not isinstance(inflight, dict):
            issues.append("metric_inflight must preserve an executable phase and stage")
        else:
            if inflight.get("phase") not in PHASES - {"plan"} or not inflight.get("stage"):
                issues.append("metric_inflight must preserve an executable phase and stage")
            if not isinstance(inflight.get("attempt"), int) or inflight["attempt"] < 1:
                issues.append("metric_inflight attempt must be a positive integer")
            if any(not isinstance(inflight.get(key), str) or not inflight[key].strip() for key in ("agent", "invocation_id")):
                issues.append("metric_inflight must preserve agent and invocation_id")
            if not valid_candidate(inflight.get("candidate")):
                issues.append("metric_inflight must preserve its candidate identity")
            elif valid_candidate(candidate) and not all(inflight["candidate"].get(key) == candidate.get(key) for key in ("commit", "fingerprint", "id")):
                issues.append("metric_inflight candidate does not match delivery-state candidate")
            if inflight.get("package") is not None:
                if inflight.get("package") not in seen:
                    issues.append(f"metric_inflight references an unknown package: {inflight.get('package')}")
                elif any(package.get("id") == inflight.get("package") and package.get("status") == "pass" for package in packages if isinstance(package, dict)):
                    issues.append(f"metric_inflight would replay completed package: {inflight.get('package')}")
    return issues


def current_state_issues(state: dict[str, Any], *, root: Path = ROOT) -> list[str]:
    issues = validate_state(state)
    if issues:
        return issues
    expected = current_candidate(root)
    if _candidate_identity(state["candidate"]) != _candidate_identity(expected):
        issues.append("delivery state candidate is stale for the current working tree")
    authority = state["authority"]
    for relative, expected_hash in authority["hashes"].items():
        try:
            normalized = _relative_path(relative, root)
            resolved = (root / normalized).resolve()
            resolved.relative_to(root.resolve())
            actual_hash = _sha256(resolved)
        except (ValueError, OSError):
            issues.append(f"delivery state authority path is unavailable: {relative}")
            continue
        if actual_hash != expected_hash:
            issues.append(f"delivery state authority hash is stale: {relative}")
    return issues


def resume_pointer(state: dict[str, Any], *, root: Path = ROOT) -> dict[str, Any]:
    issues = current_state_issues(state, root=root)
    if issues:
        raise ValueError("; ".join(issues))
    inflight = state.get("metric_inflight")
    if isinstance(inflight, dict):
        return {"resume": "inflight", **{key: inflight[key] for key in ("phase", "package", "stage", "attempt", "agent", "invocation_id", "candidate") if key in inflight}}
    pointer = state["next"]
    return {"resume": "next", **{key: pointer[key] for key in ("phase", "package", "stage") if key in pointer}}


def save_state(path: Path, state: dict[str, Any]) -> None:
    issues = validate_state(state)
    if issues:
        raise ValueError("; ".join(issues))
    path.parent.mkdir(parents=True, exist_ok=True)
    handle, temporary_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    try:
        with os.fdopen(handle, "w", encoding="utf-8", newline="\n") as stream:
            json.dump(state, stream, indent=2)
            stream.write("\n")
        os.replace(temporary_name, path)
    except Exception:
        try:
            os.unlink(temporary_name)
        except FileNotFoundError:
            pass
        raise


def load_state(path: Path) -> dict[str, Any]:
    state = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(state, dict):
        raise ValueError("delivery state must be an object")
    issues = validate_state(state)
    if issues:
        raise ValueError("; ".join(issues))
    return state


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)
    validate_parser = subparsers.add_parser("validate")
    validate_parser.add_argument("path", type=Path)
    resume_parser = subparsers.add_parser("resume")
    resume_parser.add_argument("path", type=Path)
    args = parser.parse_args()
    state = load_state(args.path)
    if args.command == "validate":
        issues = current_state_issues(state)
        payload: object = {"status": "PASS" if not issues else "BLOCKED / NOT RUN", "issues": issues}
    else:
        payload = resume_pointer(state)
    print(json.dumps(payload, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
