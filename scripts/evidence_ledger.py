#!/usr/bin/env python3
"""Stable evidence-batch contract used by independent acceptance testing."""
from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any

import sys

SCRIPTS = Path(__file__).resolve().parent
if str(SCRIPTS) not in sys.path:
    sys.path.insert(0, str(SCRIPTS))
from validation_registry import get_definition  # noqa: E402
from result_contract import current_candidate  # noqa: E402

EVIDENCE_ID = re.compile(r"^E[0-9]{2,}$")
VALID_STATUSES = {"PASS", "FAIL", "BLOCKED / NOT RUN"}
FAILURE_CLASSES = {"NONE", "TEST_FAIL", "INVOCATION_FAIL", "ENV_FAIL", "TOOL_FAIL", "FIXTURE_FAIL", "PREREQUISITE_FAIL", "TIMEOUT", "POSTCONDITION_FAIL"}
CONTEXT_FAILURE_CLASSES = {"INVOCATION_FAIL", "ENV_FAIL", "TOOL_FAIL", "FIXTURE_FAIL", "PREREQUISITE_FAIL"}
ACCEPTANCE_AGENT = "backlog_tester"


def valid_identity(value: object) -> bool:
    return isinstance(value, dict) and all(
        isinstance(value.get(key), str) and value[key].strip()
        for key in ("commit", "fingerprint", "id")
    )


def same_identity(left: object, right: object) -> bool:
    return valid_identity(left) and valid_identity(right) and all(left[key] == right[key] for key in ("commit", "fingerprint", "id"))  # type: ignore[index]


def valid_command_source(value: object) -> bool:
    if not isinstance(value, str) or not value.strip():
        return False
    if value.startswith("validation_registry:"):
        definition_id = value.split(":", 1)[1]
        if definition_id == "focused-go":
            return True
        try:
            get_definition(definition_id)
        except KeyError:
            return False
        return True
    return value.startswith("browser:")


def validate_ledger(ledger: object, required_acceptance: list[str] | None = None, expected_candidate: dict[str, Any] | None = None) -> list[str]:
    issues: list[str] = []
    if not isinstance(ledger, dict) or not valid_identity(ledger.get("candidate")):
        return ["evidence ledger requires candidate identity"]
    candidate = ledger["candidate"]
    if expected_candidate is not None and not same_identity(candidate, expected_candidate):
        issues.append("evidence ledger candidate does not match the expected exact candidate")
    batches = ledger.get("batches")
    if not isinstance(batches, list) or not batches:
        return ["evidence ledger requires at least one evidence batch"]
    ids: set[str] = set()
    covered: set[str] = set()
    for batch in batches:
        if not isinstance(batch, dict) or not isinstance(batch.get("evidence_id"), str) or not EVIDENCE_ID.fullmatch(batch["evidence_id"]):
            issues.append("each evidence batch needs a stable E## evidence_id")
            continue
        evidence_id = batch["evidence_id"]
        if evidence_id in ids:
            issues.append(f"duplicate evidence_id: {evidence_id}")
        ids.add(evidence_id)
        if not batch.get("seam"):
            issues.append(f"{evidence_id} lacks an execution seam")
        if batch.get("status") not in VALID_STATUSES:
            issues.append(f"{evidence_id} has an invalid status")
        execution = batch.get("execution")
        if not isinstance(execution, dict):
            issues.append(f"{evidence_id} requires independent execution provenance")
        else:
            if not valid_command_source(execution.get("command_source")):
                issues.append(f"{evidence_id} requires command_source provenance")
            if execution.get("preflight") not in {"PASS", "FAIL"}:
                issues.append(f"{evidence_id} has invalid preflight provenance")
            if not isinstance(execution.get("executed"), bool):
                issues.append(f"{evidence_id} requires an executed boolean")
            if execution.get("independent") is not True:
                issues.append(f"{evidence_id} must be independently executed")
            if execution.get("agent") != ACCEPTANCE_AGENT or execution.get("phase") != "test":
                issues.append(f"{evidence_id} requires independent tester provenance")
            if not isinstance(execution.get("invocation_id"), str) or not execution["invocation_id"].strip():
                issues.append(f"{evidence_id} requires an invocation_id")
            if execution.get("candidate_id") != candidate["id"]:
                issues.append(f"{evidence_id} is not bound to the ledger candidate")
            if not isinstance(execution.get("observation"), str) or not execution["observation"].strip():
                issues.append(f"{evidence_id} requires a concise observation")
            if batch.get("status") == "PASS" and (execution.get("preflight") != "PASS" or execution.get("executed") is not True):
                issues.append(f"{evidence_id} PASS is not backed by executed PASS-preflight evidence")
            if batch.get("status") == "FAIL" and (failure_class := batch.get("failure_class")) in CONTEXT_FAILURE_CLASSES | {"NONE"}:
                issues.append(f"{evidence_id} FAIL is not backed by executed non-context evidence")
            if batch.get("status") == "FAIL" and (execution.get("preflight") != "PASS" or execution.get("executed") is not True):
                issues.append(f"{evidence_id} FAIL is not backed by an executed PASS-preflight check")
            if batch.get("status") == "BLOCKED / NOT RUN" and (execution.get("preflight") != "FAIL" or execution.get("executed") is not False):
                issues.append(f"{evidence_id} BLOCKED / NOT RUN has executed or inconsistent provenance")
        failure_class = batch.get("failure_class")
        if failure_class not in FAILURE_CLASSES:
            issues.append(f"{evidence_id} requires a valid failure_class")
        elif batch.get("status") == "PASS" and failure_class != "NONE":
            issues.append(f"{evidence_id} PASS must use failure_class NONE")
        elif failure_class in CONTEXT_FAILURE_CLASSES and batch.get("status") != "BLOCKED / NOT RUN":
            issues.append(f"{evidence_id} context failure must be BLOCKED / NOT RUN")
        elif batch.get("status") == "FAIL" and failure_class in {"NONE", *CONTEXT_FAILURE_CLASSES}:
            issues.append(f"{evidence_id} FAIL must use an executed non-context failure class")
        elif batch.get("status") == "BLOCKED / NOT RUN" and failure_class not in CONTEXT_FAILURE_CLASSES:
            issues.append(f"{evidence_id} BLOCKED / NOT RUN requires a context failure class")
        rows = batch.get("acceptance_rows")
        if not isinstance(rows, list) or not rows or any(not isinstance(row, str) or not row.startswith("AC") for row in rows):
            issues.append(f"{evidence_id} must map to one or more AC rows")
        else:
            covered.update(rows)
    if required_acceptance:
        missing = sorted(set(required_acceptance) - covered)
        if missing:
            issues.append("uncovered acceptance rows: " + ", ".join(missing))
    return issues


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("ledger", type=Path)
    parser.add_argument("--expected-candidate", type=Path, help="JSON file containing the expected commit/fingerprint/id")
    parser.add_argument("--acceptance-row", action="append", default=[], help="required AC row; repeat for complete coverage")
    args = parser.parse_args()
    ledger: Any = json.loads(args.ledger.read_text(encoding="utf-8"))
    expected = current_candidate(SCRIPTS.parent)
    if args.expected_candidate:
        expected = json.loads(args.expected_candidate.read_text(encoding="utf-8"))
    issues = validate_ledger(ledger, required_acceptance=args.acceptance_row, expected_candidate=expected)
    print(json.dumps({"status": "PASS" if not issues else "FAIL", "issues": issues}, indent=2))
    return 0 if not issues else 1


if __name__ == "__main__":
    raise SystemExit(main())
