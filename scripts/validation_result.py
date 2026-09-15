#!/usr/bin/env python3
"""TestContext/failure-class extension over the shared result_contract evidence format."""
from __future__ import annotations

import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any

import result_contract as contract
from shared_delivery_contract import candidate_matches, validate_evidence_row

FAILURE_CLASSES = {
    "NONE",
    "TEST_FAIL",
    "INVOCATION_FAIL",
    "ENV_FAIL",
    "TOOL_FAIL",
    "FIXTURE_FAIL",
    "PREREQUISITE_FAIL",
    "TIMEOUT",
    "POSTCONDITION_FAIL",
}
CONTEXT_FAILURE_CLASSES = {
    "INVOCATION_FAIL",
    "ENV_FAIL",
    "TOOL_FAIL",
    "FIXTURE_FAIL",
    "PREREQUISITE_FAIL",
}
PREFLIGHT_STATES = {"PASS", "FAIL"}


@dataclass(frozen=True)
class TestContext:
    candidate_commit: str | None
    candidate_id: str | None
    candidate_fingerprint: str | None
    profile: str
    command_source: str
    command_definition: str
    working_directory: str
    prerequisites: tuple[str, ...]
    surfaces: tuple[str, ...]
    preflight: str

    def to_dict(self) -> dict[str, object]:
        payload = asdict(self)
        payload["prerequisites"] = list(self.prerequisites)
        payload["surfaces"] = list(self.surfaces)
        return payload


def make_context(root: Path, *, profile: str, definition: Any, preflight: str,
                 candidate: dict[str, object] | None = None) -> TestContext:
    candidate = candidate or contract.current_candidate(root)
    return TestContext(
        candidate_commit=str(candidate.get("commit")) if candidate.get("commit") else None,
        candidate_id=str(candidate.get("id")) if candidate.get("id") else None,
        candidate_fingerprint=str(candidate.get("fingerprint")) if candidate.get("fingerprint") else None,
        profile=profile,
        command_source=f"validation_registry:{definition.id}",
        command_definition=definition.id,
        working_directory=".",
        prerequisites=tuple(definition.prerequisites),
        surfaces=tuple(definition.surfaces),
        preflight=preflight,
    )


def _candidate_for_context(context: TestContext) -> dict[str, object]:
    payload = {
        "commit": context.candidate_commit,
        "branch": None,
        "worktree": None,
        "fingerprint": context.candidate_fingerprint,
        "id": context.candidate_id,
    }
    return payload


def result_row(*, context: TestContext, name: str, status: str, failure_class: str,
               command: list[str], started_at: str, duration_ms: int = 0,
               returncode: int | None = None, executed: bool = False,
               reason: str | None = None, stdout: object = "", stderr: object = "",
               tool: str | None = None, platform: str | None = None,
               environment: dict[str, str] | None = None) -> dict[str, object]:
    if status not in contract.VALID_RESULT_STATUSES:
        raise ValueError(f"invalid validation status: {status}")
    if failure_class not in FAILURE_CLASSES:
        raise ValueError(f"invalid failure class: {failure_class}")
    if context.preflight not in PREFLIGHT_STATES:
        raise ValueError(f"invalid preflight state: {context.preflight}")
    if status == "PASS" and failure_class != "NONE":
        raise ValueError("PASS must use failure_class NONE")
    if status == "PASS" and not executed:
        raise ValueError("PASS requires an executed check")
    if failure_class in CONTEXT_FAILURE_CLASSES and status != "BLOCKED / NOT RUN":
        raise ValueError("context failures must be BLOCKED / NOT RUN")
    if status == "FAIL" and not executed:
        raise ValueError("FAIL requires an executed check")
    if status == "BLOCKED / NOT RUN" and failure_class not in CONTEXT_FAILURE_CLASSES:
        raise ValueError("BLOCKED / NOT RUN requires a context failure class")
    if context.preflight == "FAIL" and (status != "BLOCKED / NOT RUN" or failure_class not in CONTEXT_FAILURE_CLASSES):
        raise ValueError("failed preflight requires a blocked context result")
    if context.preflight == "PASS" and failure_class in CONTEXT_FAILURE_CLASSES:
        raise ValueError("context failure requires a failed preflight")
    if failure_class in CONTEXT_FAILURE_CLASSES and executed:
        raise ValueError("context failures must not be represented as executed product evidence")
    if failure_class == "TEST_FAIL" and not executed:
        raise ValueError("TEST_FAIL requires a validly executed test")

    row = contract.result_row(
        kind="validation-check",
        name=name,
        status=status,
        command=command,
        candidate=_candidate_for_context(context),
        started_at=started_at,
        duration_ms=duration_ms,
        returncode=returncode,
        stdout=stdout,
        stderr=stderr,
        reason=reason,
        executed=executed,
        tool=tool,
        platform=platform,
        environment=environment,
    )
    row.update({
        "candidate": _candidate_for_context(context),
        "failure_class": failure_class,
        "command_source": context.command_source,
        "command_definition": context.command_definition,
        "profile": context.profile,
        "working_directory": context.working_directory,
        "prerequisites": list(context.prerequisites),
        "surfaces": list(context.surfaces),
        "preflight": context.preflight,
        "test_context": context.to_dict(),
    })
    return row


def report(*, profile: str, rows: list[dict[str, object]], root: Path,
           slice_name: str | None = None, candidate: dict[str, object] | None = None,
           reason: str | None = None) -> dict[str, object]:
    candidate = candidate or contract.current_candidate(root)
    generated_at = contract.utc_now()
    payload = {
        "schema_version": contract.SCHEMA_VERSION,
        "kind": "canonical-validation",
        "profile": profile,
        "slice": slice_name,
        "candidate": candidate,
        "evidence_id": contract.evidence_id("canonical-validation", generated_at, candidate),
        "generated_at": generated_at,
        "evidence_scope": "canonical-validation",
        "status": "FAIL" if reason else contract.overall_status(rows),
        "results": rows,
    }
    if reason:
        payload["reason"] = reason
    return payload


def validate_report(payload: object, expected_candidate: dict[str, object] | None = None) -> list[str]:
    issues: list[str] = []
    if expected_candidate is None:
        expected_candidate = contract.current_candidate(Path(__file__).resolve().parents[1])
    if not isinstance(payload, dict):
        return ["validation report is not an object"]
    if payload.get("schema_version") != contract.SCHEMA_VERSION:
        issues.append("validation schema version is invalid")
    if payload.get("kind") != "canonical-validation":
        issues.append("validation report kind is invalid")
    candidate = payload.get("candidate")
    if not isinstance(candidate, dict) or not candidate.get("commit") or not candidate.get("fingerprint") or not candidate.get("id"):
        issues.append("validation report lacks candidate identity")
    elif expected_candidate is not None and not candidate_matches(candidate, expected_candidate):
        issues.append("validation report does not match the expected exact candidate")
    if not payload.get("evidence_id") or not payload.get("generated_at"):
        issues.append("validation report lacks evidence identity")
    rows = payload.get("results")
    if not isinstance(rows, list):
        return issues + ["validation results are not a list"]
    declared_status = payload.get("status")
    expected_status = contract.overall_status(rows)
    if declared_status not in contract.VALID_RESULT_STATUSES:
        issues.append("validation report status is invalid")
    elif declared_status != expected_status and not (
        declared_status == "FAIL"
        and isinstance(payload.get("reason"), str)
        and payload["reason"].startswith("candidate changed during ")
    ):
        issues.append("validation report status contradicts its row statuses")
    for row in rows:
        if not isinstance(row, dict):
            issues.append("validation result row is not an object")
            continue
        issues.extend(validate_evidence_row(row, candidate if isinstance(candidate, dict) else {}, evidence_kind="validation-check"))
        status = row.get("status")
        failure_class = row.get("failure_class")
        executed = row.get("executed")
        if status not in contract.VALID_RESULT_STATUSES:
            issues.append("validation result status is invalid")
        if failure_class not in FAILURE_CLASSES:
            issues.append("validation failure_class is invalid")
        preflight = row.get("preflight")
        if preflight not in PREFLIGHT_STATES:
            issues.append("validation preflight state is invalid")
        if status == "PASS" and failure_class != "NONE":
            issues.append("PASS result has a failure class")
        if status == "PASS" and executed is not True:
            issues.append("PASS result is not an executed check")
        if failure_class in CONTEXT_FAILURE_CLASSES and status != "BLOCKED / NOT RUN":
            issues.append("context failure is incorrectly represented as a product failure")
        if status == "FAIL" and executed is not True:
            issues.append("FAIL result is not an executed check")
        if status == "BLOCKED / NOT RUN" and failure_class not in CONTEXT_FAILURE_CLASSES:
            issues.append("BLOCKED / NOT RUN lacks a context failure class")
        if preflight == "FAIL" and (status != "BLOCKED / NOT RUN" or failure_class not in CONTEXT_FAILURE_CLASSES):
            issues.append("failed preflight is not a blocked context result")
        if preflight == "PASS" and failure_class in CONTEXT_FAILURE_CLASSES:
            issues.append("context failure has a passing preflight")
        if failure_class in CONTEXT_FAILURE_CLASSES and executed is not False:
            issues.append("context failure is incorrectly marked executed")
        if failure_class == "TEST_FAIL" and executed is not True:
            issues.append("TEST_FAIL is not validly executed")
        if not row.get("command_source") or not row.get("command_definition"):
            issues.append("validation result lacks command provenance")
        if not candidate_matches(row.get("candidate"), candidate):
            issues.append("validation result is not bound to the report candidate")
        context = row.get("test_context")
        if not isinstance(context, dict) or not context.get("candidate_fingerprint"):
            issues.append("validation result lacks candidate-bound TestContext")
        else:
            if context.get("preflight") != preflight:
                issues.append("validation result preflight does not match its TestContext")
            if any(
                context.get(context_key) != candidate.get(candidate_key)
                for context_key, candidate_key in (("candidate_commit", "commit"), ("candidate_fingerprint", "fingerprint"), ("candidate_id", "id"))
            ):
                issues.append("validation TestContext is not bound to the report candidate")
    return issues


def dump_report(payload: dict[str, object]) -> str:
    # Validation reports can contain tool diagnostics from arbitrary locales.
    # ASCII escaping keeps structured output writeable even when a caller has
    # attached Python to a legacy cp1252 console; the JSON value remains exact.
    return json.dumps(payload, indent=2, sort_keys=False, ensure_ascii=True) + "\n"
