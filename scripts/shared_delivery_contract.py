#!/usr/bin/env python3
"""Versioned, pure delivery contracts shared by the Tockr repositories.

The file is intentionally dependency-free and is mirrored byte-for-byte in
both repositories. Repository adapters own plan discovery and command
execution; this module owns the safety semantics that must not diverge.
"""
from __future__ import annotations

import hashlib
import json
import subprocess  # nosec B404 - only fixed Git metadata commands are used
from pathlib import Path
from typing import Any, Iterable

SHARED_CONTRACT_VERSION = "1.0.0"
ROUTING_SCHEMA_VERSION = 1
EVIDENCE_SCHEMA_VERSION = 1

ALLOWED_KINDS = frozenset({"routine", "defect", "migration", "authorization", "ui", "other"})
ALLOWED_RISK_LEVELS = frozenset({"R", "E", "H"})
ALLOWED_RISK_CODES = frozenset({
    "DATA", "AUTH", "FIN", "HIST", "CONC", "OPS", "API", "UI", "PERF",
    "DEPLOY", "DEP", "GOV", "DOC",
})
MEDIUM_DISQUALIFYING_CODES = frozenset({
    "DATA", "AUTH", "FIN", "HIST", "CONC", "OPS", "API", "PERF", "DEPLOY",
    "DEP", "GOV", "DOC",
})
SCENARIOS = {
    "routine": "Routine bounded behavior change",
    "defect": "Defect correction",
    "migration": "Schema and migration change",
    "authorization": "Authorization/privacy change",
    "ui": "User-visible workflow",
}
VALID_RESULT_STATUSES = frozenset({"PASS", "FAIL", "BLOCKED / NOT RUN"})
FAILURE_CLASSES = frozenset({
    "NONE", "TEST_FAIL", "INVOCATION_FAIL", "ENV_FAIL", "TOOL_FAIL",
    "FIXTURE_FAIL", "PREREQUISITE_FAIL", "TIMEOUT", "POSTCONDITION_FAIL",
})
CONTEXT_FAILURE_CLASSES = frozenset({
    "INVOCATION_FAIL", "ENV_FAIL", "TOOL_FAIL", "FIXTURE_FAIL", "PREREQUISITE_FAIL",
})


def normalize_codes(value: str | list[Any] | None) -> list[str]:
    """Normalize a risk-code list, rejecting malformed or unknown values."""
    if value is None:
        return []
    if isinstance(value, str):
        values: Iterable[Any] = value.split(",")
    elif isinstance(value, list):
        values = value
    else:
        raise ValueError("risk codes must be a comma-separated string or list")
    cleaned: list[str] = []
    for item in values:
        if not isinstance(item, str):
            raise ValueError("risk codes must contain strings")
        code = item.strip().upper()
        if code:
            cleaned.append(code)
    codes = sorted(set(cleaned))
    unknown = sorted(set(codes) - ALLOWED_RISK_CODES)
    if unknown:
        raise ValueError(f"unknown risk code(s): {', '.join(unknown)}")
    return codes


def default_policy() -> dict[str, Any]:
    return {
        "schema_version": ROUTING_SCHEMA_VERSION,
        "conceptual_role": "backlog_implementer",
        "default_effort": "high",
        "high_runtime_agent": "backlog_implementer",
        "medium_runtime_agent": "backlog_implementer_medium",
        "approved_rules": [],
    }


def _read_json(path: Path) -> object | None:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None


def _validated_policy(value: object) -> tuple[dict[str, Any], list[str]]:
    """Normalize policy data conservatively for both file and in-memory callers."""
    warnings: list[str] = []
    required = default_policy()
    if not isinstance(value, dict) or any(value.get(key) != required[key] for key in (
        "schema_version", "conceptual_role", "default_effort", "high_runtime_agent",
        "medium_runtime_agent",
    )):
        return required, ["routing policy is missing, unreadable, or has an unsupported default"]
    rules = value.get("approved_rules", [])
    if not isinstance(rules, list):
        return {**required, "approved_rules": []}, ["approved_rules is not a list; medium authorization was discarded"]

    valid: list[dict[str, Any]] = []
    for index, rule in enumerate(rules):
        if not isinstance(rule, dict):
            warnings.append(f"approved_rules[{index}] is not an object and was ignored")
            continue
        if "risk_codes" not in rule or not isinstance(rule.get("risk_codes"), list):
            warnings.append(f"approved_rules[{index}] lacks a list-valued risk_codes field and was ignored")
            continue
        try:
            codes = normalize_codes(rule.get("risk_codes"))
        except ValueError as error:
            warnings.append(f"approved_rules[{index}] is malformed ({error}) and was ignored")
            continue
        kinds = rule.get("work_kinds")
        if (
            not isinstance(rule.get("id"), str) or not rule["id"].strip()
            or rule.get("enabled") is not True
            or rule.get("risk_level") not in ALLOWED_RISK_LEVELS
            or not isinstance(kinds, list) or not kinds
            or any(not isinstance(kind, str) or kind not in ALLOWED_KINDS for kind in kinds)
            or not isinstance(rule.get("evidence_gate_run_id"), str)
            or not rule["evidence_gate_run_id"].strip()
        ):
            warnings.append(f"approved_rules[{index}] is incomplete and was ignored")
            continue
        normalized = dict(rule)
        normalized["risk_codes"] = codes
        normalized["work_kinds"] = list(dict.fromkeys(kinds))
        valid.append(normalized)
    policy = dict(required)
    policy["approved_rules"] = valid
    return policy, warnings


def load_policy(path: Path) -> tuple[dict[str, Any], list[str]]:
    """Load policy conservatively; malformed entries are discarded, never relaxed."""
    return _validated_policy(_read_json(path))


REQUIRED_GATE_SCENARIOS = frozenset({"Routine bounded behavior change", "Defect correction"})
INTEGRITY_GATE_SCENARIOS = frozenset({"Schema and migration change", "Authorization/privacy change"})
EFFICIENCY_METRICS = frozenset({"input_tokens", "output_tokens", "tool_calls", "elapsed_seconds", "estimated_cost_usd"})


def _eval_report(path_value: object, report_root: Path | None, scenario: str, configuration: str) -> bool:
    if not isinstance(path_value, str) or not path_value.strip():
        return False
    path = Path(path_value)
    if not path.is_file() and report_root is not None:
        path = report_root / path.name
    value = _read_json(path)
    if not isinstance(value, dict):
        return False
    if value.get("schema_version") not in {1, 2} or not isinstance(value.get("run_id"), str) or not value["run_id"].strip():
        return False
    if value.get("scenario") != scenario or value.get("configuration") != configuration:
        return False
    quality = value.get("quality")
    implementation = value.get("implementation")
    if not isinstance(quality, dict) or any(quality.get(key) != "Pass" for key in ("outcome", "review_verdict", "test_verdict")):
        return False
    if not isinstance(implementation, dict) or not isinstance(implementation.get("head"), str) or not implementation["head"].strip() or not isinstance(implementation.get("diff_sha256"), str) or len(implementation["diff_sha256"]) != 64:
        return False
    return True


def gate_supports(manifest: object, scenario: str | None, *, report_root: Path | None = None) -> bool:
    """Validate the complete paired-evaluation gate before medium authorization."""
    if not isinstance(manifest, dict):
        return False
    if (
        manifest.get("schema_version") != 1
        or not isinstance(manifest.get("run_id"), str)
        or not manifest["run_id"].strip()
        or not isinstance(manifest.get("created_at_utc"), str)
        or manifest.get("baseline_configuration") != "implementer high"
        or manifest.get("candidate_configuration") != "implementer medium"
        or manifest.get("result") != "Adopt medium"
        or manifest.get("required_scenarios_missing") != []
    ):
        return False
    threshold = manifest.get("minimum_improvement_percent")
    if not isinstance(threshold, (int, float)) or threshold <= 0:
        return False
    rows = manifest.get("scenario_results")
    if not isinstance(rows, list) or not rows:
        return False
    by_scenario: dict[str, dict[str, Any]] = {}
    for row in rows:
        if not isinstance(row, dict) or not isinstance(row.get("scenario"), str) or row["scenario"] in by_scenario:
            return False
        if row.get("verdict") != "Pass" or row.get("best_metric") not in EFFICIENCY_METRICS:
            return False
        gain = row.get("best_improvement_percent")
        if not isinstance(gain, (int, float)) or gain < threshold:
            return False
        if not _eval_report(row.get("baseline_report"), report_root, row["scenario"], "baseline high"):
            return False
        if not _eval_report(row.get("candidate_report"), report_root, row["scenario"], "candidate medium"):
            return False
        by_scenario[row["scenario"]] = row
    if not REQUIRED_GATE_SCENARIOS.issubset(by_scenario) or not INTEGRITY_GATE_SCENARIOS.intersection(by_scenario):
        return False
    if scenario is None:
        return True
    return scenario in by_scenario


def route_work(
    *,
    work_package: str,
    work_kind: str,
    risk_level: str,
    risk_codes: str | list[Any] | None,
    policy: dict[str, Any] | None = None,
    supporting_gate: bool = False,
    approved_gate_ids: Iterable[str] = (),
    plan_route: dict[str, object] | None = None,
    plan_issues: Iterable[str] = (),
    observed_kind: str | None = None,
    observed_risk_level: str | None = None,
    observed_risk_codes: str | list[Any] | None = None,
    history: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Return a conservative routing decision for one exact work package."""
    reasons: list[str] = []
    issues = list(plan_issues)
    active_policy, policy_warnings = _validated_policy(policy if policy is not None else default_policy())
    try:
        codes = normalize_codes(risk_codes)
    except ValueError as error:
        codes = []
        issues.append(str(error))
    if work_kind not in ALLOWED_KINDS:
        issues.append(f"unsupported work kind: {work_kind}")
    if risk_level not in ALLOWED_RISK_LEVELS:
        issues.append(f"unsupported risk level: {risk_level}")
    if plan_route is not None:
        if (
            plan_route.get("kind") != work_kind
            or plan_route.get("risk") != risk_level
            or list(plan_route.get("risk_codes", [])) != codes
        ):
            issues.append("supplied routing metadata does not match the plan-bound Route signature")

    escalated = False
    effective_codes = codes
    if observed_risk_level is not None or observed_risk_codes is not None:
        try:
            observed_codes = normalize_codes(observed_risk_codes)
        except ValueError as error:
            observed_codes = []
            issues.append(f"observed routing metadata is malformed: {error}")
            escalated = True
        if observed_risk_level is not None and observed_risk_level not in ALLOWED_RISK_LEVELS:
            issues.append(f"observed risk level is unsupported: {observed_risk_level}")
            escalated = True
        elif observed_risk_level is not None:
            levels = {"R": 0, "E": 1, "H": 2}
            escalated = escalated or levels[observed_risk_level] > levels.get(risk_level, 2)
            escalated = escalated or bool(set(observed_codes) - set(codes))
            effective_codes = sorted(set(codes) | set(observed_codes))
        else:
            escalated = escalated or bool(set(observed_codes) - set(codes))
            effective_codes = sorted(set(codes) | set(observed_codes))
        if observed_kind is not None and observed_kind != work_kind:
            escalated = True

    recommendation = "high"
    recommendation_state = "keep-high"
    if issues:
        reasons.append("routing validation failed: " + "; ".join(issues))
    elif escalated:
        reasons.append("observed work risk increased; medium execution must escalate before mutation")
    elif work_kind == "defect":
        reasons.append("defect repairs always use high reasoning")
    elif risk_level in {"E", "H"}:
        reasons.append("E/H work remains on high reasoning")
    elif work_kind != "routine":
        reasons.append("only routine Risk R work is eligible for medium reasoning")
    elif set(codes) & MEDIUM_DISQUALIFYING_CODES:
        reasons.append("Risk R includes a medium-disqualifying surface: " + ",".join(codes))
    elif not supporting_gate:
        reasons.append("no successful Adopt medium paired-evaluation gate supports the work kind")
    else:
        recommendation = "medium"
        recommendation_state = "adopt-medium-candidate"
        reasons.append("paired-evaluation gate supports routine Risk R routing")

    approved_rule: dict[str, Any] | None = None
    gate_ids = set(approved_gate_ids)
    if recommendation == "medium":
        for rule in active_policy.get("approved_rules", []):
            if (
                isinstance(rule, dict)
                and rule.get("enabled") is True
                and rule.get("risk_level") == risk_level
                and work_kind in rule.get("work_kinds", [])
                and rule.get("risk_codes") == codes
                and rule.get("evidence_gate_run_id") in gate_ids
            ):
                approved_rule = rule
                break

    authorized = "high"
    approval_required = recommendation == "medium"
    if approved_rule is not None and not escalated:
        authorized = "medium"
        approval_required = False
        reasons.append(f"enabled policy rule '{approved_rule.get('id')}' cites a locally verifiable Adopt medium gate")
    elif recommendation == "medium":
        reasons.append("medium is recommended but not authorized by an enabled human-approved policy rule")

    if escalated:
        authorized = "high"
        approval_required = False
    return {
        "contract_version": SHARED_CONTRACT_VERSION,
        "schema_version": ROUTING_SCHEMA_VERSION,
        "conceptual_role": "backlog_implementer",
        "work_package": work_package,
        "work_kind": work_kind,
        "risk_level": risk_level,
        "risk_codes": effective_codes,
        "scenario": SCENARIOS.get(work_kind),
        "recommendation": recommendation,
        "recommendation_state": recommendation_state,
        "authorized_effort": authorized,
        "runtime_agent": "backlog_implementer_medium" if authorized == "medium" else "backlog_implementer",
        "approval_required": approval_required,
        "routing_escalation": "high" if escalated else None,
        "policy_warnings": policy_warnings,
        "plan_routing_issues": issues,
        "history": history or {},
        "reason": "; ".join(reasons),
    }


def _git(root: Path, *arguments: str) -> str | None:
    try:
        completed = subprocess.run(
            ["git", "-C", str(root), *arguments],
            text=True,
            capture_output=True,
            encoding="utf-8",
            errors="replace",
            check=True,
        )  # nosec B603 - fixed metadata command
    except (OSError, subprocess.CalledProcessError):
        return None
    return completed.stdout.strip() or None


def _candidate_paths(root: Path) -> list[str]:
    changed = (_git(root, "diff", "--name-only", "HEAD") or "").splitlines()
    untracked = (_git(root, "ls-files", "--others", "--exclude-standard") or "").splitlines()
    ignored_prefixes = ("docs/implementation/audits/", ".codex/agent-evals/", "__pycache__/", "scripts/__pycache__/")
    paths: list[str] = []
    for path in (*changed, *untracked):
        normalized = path.replace("\\", "/")
        if normalized.startswith(ignored_prefixes):
            continue
        if normalized.startswith(".codex/delivery-state/") and normalized != ".codex/delivery-state/README.md":
            continue
        if normalized == ".codex/delivery-state/README.md" and not _git(root, "ls-files", "--error-unmatch", "--", normalized):
            continue
        if normalized not in paths:
            paths.append(normalized)
    if _git(root, "ls-files", "--error-unmatch", "--", ".codex/delivery-state/README.md") and ".codex/delivery-state/README.md" not in paths:
        paths.append(".codex/delivery-state/README.md")
    return sorted(paths)


def working_tree_fingerprint(root: Path) -> str:
    paths = _candidate_paths(root)
    status = _git(root, "status", "--porcelain", "--untracked-files=no") or ""
    digest = hashlib.sha256(f"status:{status}".encode("utf-8"))
    for relative in paths:
        path = root / relative
        digest.update(relative.encode("utf-8"))
        digest.update(b"\0")
        try:
            digest.update(path.read_bytes())
        except OSError:
            digest.update(b"<missing>")
        digest.update(b"\0")
    return digest.hexdigest()


def current_candidate(root: Path) -> dict[str, object]:
    commit = _git(root, "rev-parse", "HEAD")
    branch = _git(root, "branch", "--show-current")
    fingerprint = working_tree_fingerprint(root)
    identity = hashlib.sha256(f"{commit or 'unknown'}\0{fingerprint}".encode("utf-8")).hexdigest()
    return {
        "commit": commit,
        "branch": branch,
        "worktree": "dirty" if _git(root, "status", "--porcelain") else "clean",
        "fingerprint": fingerprint,
        "id": identity,
    }


def candidate_matches(left: object, right: object) -> bool:
    return isinstance(left, dict) and isinstance(right, dict) and all(
        left.get(key) == right.get(key) for key in ("commit", "fingerprint", "id")
    )


def evidence_id(kind: str, name: str, candidate: dict[str, object]) -> str:
    digest = hashlib.sha256(json.dumps({
        "contract_version": SHARED_CONTRACT_VERSION,
        "kind": kind,
        "name": name,
        "candidate": {key: candidate.get(key) for key in ("commit", "fingerprint", "id")},
    }, sort_keys=True, separators=(",", ":")).encode("utf-8")).hexdigest()[:24]
    return f"{kind}:{name}:{digest}"


def overall_status(rows: list[dict[str, object]]) -> str:
    if not rows:
        return "BLOCKED / NOT RUN"
    if any(not isinstance(row, dict) for row in rows):
        return "FAIL"
    if any(row.get("status") == "FAIL" for row in rows):
        return "FAIL"
    if any(row.get("status") == "BLOCKED / NOT RUN" for row in rows):
        return "BLOCKED / NOT RUN"
    return "PASS"


def evidence_row(
    *,
    name: str,
    status: str,
    candidate: dict[str, object],
    executed: bool,
    failure_class: str = "NONE",
    command: list[str] | None = None,
    reason: str | None = None,
    evidence_scope: str = "canonical-validation",
) -> dict[str, object]:
    if status not in VALID_RESULT_STATUSES:
        raise ValueError(f"invalid evidence status: {status}")
    if failure_class not in FAILURE_CLASSES:
        raise ValueError(f"invalid failure class: {failure_class}")
    if status == "PASS" and (not executed or failure_class != "NONE"):
        raise ValueError("PASS requires an executed check with failure_class NONE")
    if status == "FAIL" and (not executed or failure_class == "NONE" or failure_class in CONTEXT_FAILURE_CLASSES):
        raise ValueError("FAIL requires executed non-context failure evidence")
    if status == "BLOCKED / NOT RUN" and (executed or failure_class not in CONTEXT_FAILURE_CLASSES):
        raise ValueError("BLOCKED / NOT RUN requires an unexecuted context failure")
    row: dict[str, object] = {
        "id": name,
        "name": name,
        "status": status,
        "executed": executed,
        "failure_class": failure_class,
        "candidate": dict(candidate),
        "evidence_id": evidence_id("validation-row", name, candidate),
        "evidence_scope": evidence_scope,
        "command": list(command or []),
    }
    if reason:
        row["reason"] = reason
    return row


def evidence_report(
    *,
    kind: str,
    candidate: dict[str, object],
    rows: list[dict[str, object]],
    report_name: str | None = None,
    reason: str | None = None,
) -> dict[str, object]:
    identity_name = report_name or kind
    payload: dict[str, object] = {
        "contract_version": SHARED_CONTRACT_VERSION,
        "schema_version": EVIDENCE_SCHEMA_VERSION,
        "kind": kind,
        "candidate": dict(candidate),
        "evidence_id": evidence_id("validation-report", identity_name, candidate),
        "status": overall_status(rows),
        "results": rows,
    }
    if report_name:
        payload["report_name"] = report_name
    if reason:
        payload["reason"] = reason
    return payload


def validate_evidence_row(row: object, expected_candidate: dict[str, object], *, evidence_kind: str = "validation-row") -> list[str]:
    """Validate the shared state, candidate and evidence identity of one row."""
    if not isinstance(row, dict):
        return ["evidence row is not an object"]
    issues: list[str] = []
    status = row.get("status")
    failure_class = row.get("failure_class")
    if status not in VALID_RESULT_STATUSES:
        issues.append(f"evidence row {row.get('name')} status is invalid")
    if failure_class not in FAILURE_CLASSES:
        issues.append(f"evidence row {row.get('name')} failure_class is invalid")
    candidate = row.get("candidate")
    if not candidate_matches(candidate, expected_candidate):
        issues.append(f"evidence row {row.get('name')} is bound to another candidate")
    name = row.get("name")
    if not isinstance(name, str) or not name.strip():
        issues.append("evidence row lacks a stable name")
    elif isinstance(candidate, dict) and row.get("evidence_id") != evidence_id(evidence_kind, name, candidate):
        issues.append(f"evidence row {name} has an invalid evidence identity")
    if status == "PASS" and (row.get("executed") is not True or failure_class != "NONE"):
        issues.append(f"PASS row {name} lacks executed/NONE proof")
    if status == "FAIL" and (row.get("executed") is not True or failure_class == "NONE" or failure_class in CONTEXT_FAILURE_CLASSES):
        issues.append(f"FAIL row {name} lacks executed non-context proof")
    if status == "BLOCKED / NOT RUN" and (row.get("executed") is True or failure_class not in CONTEXT_FAILURE_CLASSES):
        issues.append(f"blocked row {name} lacks unexecuted context proof")
    return issues


def validate_evidence_report(payload: object, expected_candidate: dict[str, object]) -> list[str]:
    issues: list[str] = []
    if not isinstance(payload, dict):
        return ["evidence report is not an object"]
    if payload.get("contract_version") != SHARED_CONTRACT_VERSION:
        issues.append("evidence contract version is invalid")
    if payload.get("schema_version") != EVIDENCE_SCHEMA_VERSION:
        issues.append("evidence schema version is invalid")
    candidate = payload.get("candidate")
    if not candidate_matches(candidate, expected_candidate):
        issues.append("evidence report is not bound to the exact candidate")
    if isinstance(candidate, dict) and payload.get("evidence_id") != evidence_id("validation-report", str(payload.get("report_name") or payload.get("kind")), candidate):
        issues.append("evidence report identity is not bound to its candidate")
    rows = payload.get("results")
    if not isinstance(rows, list) or not rows:
        return issues + ["evidence report results must be a non-empty list"]
    if payload.get("status") not in VALID_RESULT_STATUSES:
        issues.append("evidence report status is invalid")
    elif payload.get("status") != overall_status([row for row in rows if isinstance(row, dict)]):
        issues.append("evidence report status contradicts row statuses")
    for row in rows:
        issues.extend(validate_evidence_row(row, candidate if isinstance(candidate, dict) else {}))
    return issues


def blocked_result(
    *,
    name: str,
    candidate: dict[str, object],
    reason: str,
    failure_class: str = "PREREQUISITE_FAIL",
) -> dict[str, object]:
    return evidence_row(
        name=name,
        status="BLOCKED / NOT RUN",
        candidate=candidate,
        executed=False,
        failure_class=failure_class,
        reason=reason,
    )
