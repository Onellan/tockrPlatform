#!/usr/bin/env python3
"""Shared, publish-safe contract for repository-local evidence results."""
from __future__ import annotations

import json
import re
import shlex
from datetime import datetime, timezone
from pathlib import Path

from shared_delivery_contract import _candidate_paths, _git, current_candidate, evidence_id, overall_status, working_tree_fingerprint

SCHEMA_VERSION = 2
VALID_RESULT_STATUSES = {"PASS", "FAIL", "BLOCKED / NOT RUN"}
MAX_OUTPUT_CHARS = 4000
MAX_OUTPUT_BYTES = 4096
MAX_RESULT_ROWS = 256
MAX_RESULT_FILE_BYTES = 1024 * 1024
MAX_ARTIFACT_FILE_BYTES = 256 * 1024
MAX_ARTIFACT_TOTAL_BYTES = 1024 * 1024

_SECRET_ASSIGNMENT_PATTERN = re.compile(r"""(?i)((["']?)(?:password|passwd|passphrase|token|access[_-]?token|secret|client[_-]?secret|api[_-]?key|authorization|cookie)\2\s*[:=]\s*)("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^,\s};]+)""")
_SECRET_FLAG_NAME_PATTERN = re.compile(r"(?i)^--(?:password|passwd|passphrase|token|access[-_]token|secret|client[-_]secret|api[-_]key|authorization|cookie)$")
_SECRET_PATTERNS = (
    re.compile(r"(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+"),
    re.compile(r"(?i)--(?:password|passwd|passphrase|token|access[-_]token|secret|client[-_]secret|api[-_]key|authorization|cookie)(?:=|\s+)[^\s]+"),
    _SECRET_ASSIGNMENT_PATTERN,
    re.compile(r"-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----", re.DOTALL),
)
_GOVERNED_PATTERNS = (
    re.compile(r"\bJOURNEY-[A-Z0-9_-]+\b"),
    re.compile(r"\bJRN-[A-Z0-9_-]+\b"),
)
_LOCAL_PATH_PATTERNS = (
    re.compile(r'''(?i)(?:[A-Z]:\\|\\\\)[^:\r\n,;}\]\)"]+'''),
    re.compile(r"(?i)(?:[A-Z]:\\|\\\\)[^\s\r\n]+"),
    re.compile(r'''(?i)\b[A-Z]:\\(?=$|[\s|,;}\]\)"])'''),
    re.compile(r"(?<![A-Za-z0-9])(?:/home/|/Users/|/workspace/|/tmp/|/var/tmp/|/mnt/)[^\s\r\n]+"),
)
_SECRET_KEY_NAMES = {
    "password",
    "passwd",
    "passphrase",
    "token",
    "accesstoken",
    "secret",
    "clientsecret",
    "apikey",
    "authorization",
    "cookie",
}
_SENSITIVE_CONTENT_KEY_PARTS = (
    "document",
    "bytes",
    "raw",
    "log",
    "stdout",
    "stderr",
    "trace",
    "stack",
    "freetext",
    "governedfree",
    "governedtext",
)
_RESULT_PAYLOAD_KEYS = (
    "schema_version", "kind", "slice", "candidate", "evidence_id", "generated_at",
    "evidence_scope", "status", "results", "authentication", "preflight", "prerequisite_probe", "reason",
)
_RESULT_ROW_KEYS = (
    "id", "name", "status", "executed", "execution_mode", "candidate", "command", "command_text",
    "returncode", "outcome", "started_at", "duration_ms", "evidence_id", "evidence_path",
    "evidence_type", "acceptance_id", "diagnostic", "adapter", "platform", "environment",
    "tool", "tool_version", "tool_version_command", "scope", "postcondition_commands",
)
_CANDIDATE_KEYS = ("commit", "branch", "worktree", "fingerprint", "id")
_AUTHENTICATION_KEYS = ("mode", "roles")
_SAFE_SLICE_PATTERN = re.compile(r"^(?:B\d+|T\d+|P[XYZ]-[A-Z]\d+)-S\d+$")
_SAFE_BRANCH_PATTERN = re.compile(r"^[A-Za-z0-9_./-]{1,128}$")
_SAFE_EVIDENCE_PATTERN = re.compile(r"^[A-Za-z0-9:_-]{1,256}$")
STANDARDS_RESULT_NAMES = {
    "gofmt", "go-test", "go-vet", "docs", "architecture", "imports", "portability", "repository-hygiene", "diff-check", "gosec", "govulncheck",
}

# This is the authoritative full-profile matrix.  The coordinator may choose a
# Windows or POSIX executable at the tooling boundary, but it must emit exactly
# one row for each name below.  Keep this sequence stable so reports are easy to
# compare while keeping validation order-independent for imported evidence.
FULL_PROFILE_RESULT_ORDER = (
    "gofmt",
    "go-test",
    "go-vet",
    "docs",
    "architecture",
    "imports",
    "portability",
    "repository-hygiene",
    "diff-check",
    "gosec",
    "govulncheck",
    "script-tests",
    "module-tidy",
    "go-build-amd64",
    "go-build-arm64-cross",
    "go-test-race-sqlite",
    "go-test-race-http",
    "migration-fresh-upgrade",
    "gitleaks",
    "osv-scanner",
    "docker-build-amd64",
    "docker-build-arm64",
    "docker-compose-contract",
    "docker-runtime-amd64",
    "docker-runtime-arm64",
)
FULL_PROFILE_RESULT_NAMES = set(FULL_PROFILE_RESULT_ORDER)
FULL_PROFILE_ROW_KIND = "full-check"
FULL_PROFILE_KIND = "full-validation-profile"
FULL_PROFILE_EVIDENCE_SCOPE = "full-local-validation"
FULL_PROFILE_ADAPTERS = {
    "process",
    "module-tidy",
    "go-build",
    "docker-build",
    "compose-contract",
    "docker-runtime",
}
FULL_PROFILE_CHECK_ADAPTERS = {
    **{name: "process" for name in STANDARDS_RESULT_NAMES},
    "script-tests": "process",
    "module-tidy": "module-tidy",
    "go-build-amd64": "go-build",
    "go-build-arm64-cross": "go-build",
    "go-test-race-sqlite": "process",
    "go-test-race-http": "process",
    "migration-fresh-upgrade": "process",
    "gitleaks": "process",
    "osv-scanner": "process",
    "docker-build-amd64": "docker-build",
    "docker-build-arm64": "docker-build",
    "docker-compose-contract": "compose-contract",
    "docker-runtime-amd64": "docker-runtime",
    "docker-runtime-arm64": "docker-runtime",
}
FULL_PROFILE_CHECK_PLATFORMS = {
    "go-build-arm64-cross": "linux/arm64",
    "docker-build-amd64": "linux/amd64",
    "docker-build-arm64": "linux/arm64",
    "docker-runtime-amd64": "linux/amd64",
    "docker-runtime-arm64": "linux/arm64",
}
FULL_PROFILE_CHECK_ENVIRONMENTS = {
    "go-build-arm64-cross": {"GOOS": "linux", "GOARCH": "arm64", "CGO_ENABLED": "0"},
}
FULL_PROFILE_SCANNER_TOOLS = {
    "gosec": "gosec",
    "govulncheck": "govulncheck",
    "gitleaks": "gitleaks",
    "osv-scanner": "osv-scanner",
}
FULL_PROFILE_SCANNER_VERSION_COMMANDS = {
    "gosec": ["gosec", "-version"],
    "govulncheck": ["govulncheck", "-version"],
    "gitleaks": ["gitleaks", "--version"],
    "osv-scanner": ["osv-scanner", "--version"],
}
FULL_PROFILE_SCANNER_SCOPES = {
    "gosec": "Go packages ./...",
    "govulncheck": "Go packages ./...",
    "gitleaks": "repository source and Git history",
    "osv-scanner": "repository source tree",
}
BROWSER_RESULT_NAMES = {
    *(f"J{number}" for number in range(1, 14)),
    "ROLE-manager", "ROLE-contributor", "ROLE-viewer",
    "AUTH-ALLOW", "AUTH-NEGATIVE", "AUTH-SCOPE-NEGATIVE", "CSRF-NEGATIVE", "RETURN-CONTEXT",
    "UI-KEYBOARD-NARROW", "UI-NO-JAVASCRIPT", *(f"AC-A2-S01-0{number}" for number in range(1, 7)),
    "AUTHENTICATED-COVERAGE",
    "PY-Y1-S02",
    *(f"PY-Y2-S0{number}" for number in range(1, 6)),
    *(f"PY-Y3-S0{number}" for number in range(1, 8)),
    "PY-Y4-S01", "PY-Y4-S02", "PY-Y4-S03", "PY-Y4-S05",
}


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")


def _redact_plain_text(text: str) -> str:
    redacted = text
    for pattern in _SECRET_PATTERNS:
        if pattern is _SECRET_ASSIGNMENT_PATTERN:
            redacted = pattern.sub(lambda match: f'{match.group(1)}"[redacted secret]"', redacted)
        elif pattern.groups:
            redacted = pattern.sub("[redacted secret]", redacted)
        else:
            redacted = pattern.sub("[redacted secret]", redacted)
    for pattern in _GOVERNED_PATTERNS:
        redacted = pattern.sub("[redacted governed identifier]", redacted)
    for pattern in _LOCAL_PATH_PATTERNS:
        redacted = pattern.sub("[redacted local path]", redacted)
    return redacted


def _bounded_text(text: str, limit: int, *, structured: bool = False) -> str:
    marker = "…[truncated]"
    if len(text) > limit or len(text.encode("utf-8", errors="replace")) > MAX_OUTPUT_BYTES:
        if structured:
            return json.dumps({"truncated": True, "reason": "structured output exceeded bound"})
        budget = min(limit, MAX_OUTPUT_BYTES)
        marker_bytes = len(marker.encode("utf-8"))
        prefix = text[: max(0, budget - len(marker))]
        while len(prefix.encode("utf-8", errors="replace")) + marker_bytes > MAX_OUTPUT_BYTES:
            prefix = prefix[:-1]
        return prefix + marker
    return text


def redact_text(value: object, limit: int = MAX_OUTPUT_CHARS) -> str:
    """Bound diagnostics and remove credential/governed-text patterns."""
    text = "" if value is None else str(value)
    structured = False
    try:
        parsed = json.loads(text)
    except (TypeError, ValueError):
        redacted = _redact_plain_text(text)
    else:
        if isinstance(parsed, (dict, list)):
            structured = True
            redacted = json.dumps(sanitize_value(parsed), ensure_ascii=True)
        else:
            redacted = _redact_plain_text(text)
    return _bounded_text(redacted, limit, structured=structured)


def _redact_command_parts(command: object) -> list[str]:
    if not isinstance(command, list):
        return []
    redacted: list[str] = []
    redact_next = False
    for part in command:
        if not isinstance(part, str) or not part:
            redacted.append("[redacted command value]")
            redact_next = False
            continue
        if redact_next and not part.startswith("-"):
            redacted.append("[redacted secret]")
            redact_next = False
            continue
        redact_next = bool(_SECRET_FLAG_NAME_PATTERN.fullmatch(part))
        redacted.append(redact_text(part, limit=256))
    return redacted


def sanitize_value(value: object) -> object:
    """Recursively sanitize imported evidence before it is rendered by an audit."""
    if isinstance(value, dict):
        sanitized: dict[str, object] = {}
        for key, item in value.items():
            key_text = str(key)
            normalized_key = re.sub(r"[^a-z0-9]", "", key_text.strip("\"'").lower())
            if normalized_key in _SECRET_KEY_NAMES:
                sanitized[key_text] = "[redacted secret]"
            elif any(part in normalized_key for part in _SENSITIVE_CONTENT_KEY_PARTS):
                sanitized[key_text] = "[redacted sensitive content]"
            else:
                sanitized[key_text] = sanitize_value(item)
        return sanitized
    if isinstance(value, list):
        return [sanitize_value(item) for item in value]
    if isinstance(value, str):
        return _redact_plain_text(value)
    return value


def sanitize_result_payload(value: object) -> object:
    """Keep imported evidence to the supported schema and omit raw diagnostics/content."""
    sanitized = sanitize_value(value)
    if not isinstance(sanitized, dict):
        return sanitized
    projected = {key: sanitized[key] for key in _RESULT_PAYLOAD_KEYS if key in sanitized}
    if not isinstance(projected.get("kind"), str) or projected["kind"] not in {"standards-run", "full-validation-profile", "browser-certification", "browser-certification-prerequisite-probe"}:
        projected["kind"] = "[redacted result kind]"
    if "slice" in projected and (not isinstance(projected["slice"], str) or not _SAFE_SLICE_PATTERN.fullmatch(projected["slice"])):
        projected["slice"] = "[redacted slice]"
    if "evidence_id" in projected and (not isinstance(projected["evidence_id"], str) or not _SAFE_EVIDENCE_PATTERN.fullmatch(projected["evidence_id"])):
        projected["evidence_id"] = "[redacted evidence identity]"
    if "generated_at" in projected and (not isinstance(projected["generated_at"], str) or not re.fullmatch(r"\d{4}-\d{2}-\d{2}T[0-9:.+-]+Z", projected["generated_at"])):
        projected["generated_at"] = "[redacted timestamp]"
    if "evidence_scope" in projected and (not isinstance(projected["evidence_scope"], str) or not re.fullmatch(r"[a-z0-9-]{1,64}", projected["evidence_scope"])):
        projected["evidence_scope"] = "[redacted evidence scope]"
    if not isinstance(projected.get("status"), str) or projected["status"] not in VALID_RESULT_STATUSES:
        projected["status"] = "FAIL"
    candidate = projected.get("candidate")
    if isinstance(candidate, dict):
        safe_candidate: dict[str, object] = {}
        for key in _CANDIDATE_KEYS:
            if key not in candidate:
                continue
            item = candidate[key]
            if key == "commit" and isinstance(item, str) and re.fullmatch(r"[0-9a-f]{40}", item):
                safe_candidate[key] = item
            elif key in {"fingerprint", "id"} and isinstance(item, str) and re.fullmatch(r"[0-9a-f]{64}", item):
                safe_candidate[key] = item
            elif key in {"branch", "worktree"} and (item is None or item == ""):
                # Detached checkouts legitimately have no branch name. Preserve
                # that state so the top-level candidate and every row retain
                # the same candidate identity after projection.
                safe_candidate[key] = item
            elif key in {"branch", "worktree"} and isinstance(item, str) and _SAFE_BRANCH_PATTERN.fullmatch(item):
                safe_candidate[key] = item
            else:
                safe_candidate[key] = "[redacted candidate value]"
        projected["candidate"] = safe_candidate
    elif "candidate" in projected:
        projected["candidate"] = {}
    authentication = projected.get("authentication")
    if isinstance(authentication, dict):
        safe_authentication = {key: authentication[key] for key in _AUTHENTICATION_KEYS if key in authentication}
        if not isinstance(safe_authentication.get("mode"), str) or not re.fullmatch(r"[a-z0-9-]{1,64}", safe_authentication["mode"]):
            if "mode" in safe_authentication:
                safe_authentication["mode"] = "[redacted authentication mode]"
        roles = safe_authentication.get("roles")
        if isinstance(roles, list):
            safe_authentication["roles"] = [
                role if isinstance(role, str) and re.fullmatch(r"[a-z0-9_-]{1,64}", role) else "[redacted role]"
                for role in roles
            ]
        elif "roles" in safe_authentication:
            safe_authentication["roles"] = []
        projected["authentication"] = safe_authentication
    elif "authentication" in projected:
        projected["authentication"] = {}
    rows = projected.get("results")
    if isinstance(rows, list):
        projected_rows: list[object] = []
        if projected.get("kind") == "standards-run":
            allowed_names = STANDARDS_RESULT_NAMES
        elif projected.get("kind") == "full-validation-profile":
            allowed_names = FULL_PROFILE_RESULT_NAMES
        elif projected.get("kind") in {"browser-certification", "browser-certification-prerequisite-probe"}:
            allowed_names = BROWSER_RESULT_NAMES
        else:
            allowed_names = set()
        for row in rows:
            if not isinstance(row, dict):
                projected_rows.append(None)
                continue
            projected_row = {key: row[key] for key in _RESULT_ROW_KEYS if key in row}
            if "candidate" in projected_row:
                row_candidate = projected_row["candidate"]
                if isinstance(row_candidate, dict):
                    projected_row["candidate"] = {
                        key: row_candidate[key] for key in _CANDIDATE_KEYS if key in row_candidate
                    }
                else:
                    projected_row["candidate"] = {}
            if not isinstance(projected_row.get("name"), str) or projected_row["name"] not in allowed_names:
                projected_row["name"] = "[redacted result name]"
                if "acceptance_id" in projected_row:
                    projected_row["acceptance_id"] = "[redacted result name]"
            if "diagnostic" in projected_row:
                projected_row["diagnostic"] = redact_text(projected_row["diagnostic"], limit=MAX_OUTPUT_CHARS)
            if "adapter" in projected_row and (
                not isinstance(projected_row["adapter"], str)
                or not re.fullmatch(r"[a-z0-9-]{1,64}", projected_row["adapter"])
            ):
                projected_row["adapter"] = "[redacted adapter]"
            if "platform" in projected_row and (
                not isinstance(projected_row["platform"], str)
                or not re.fullmatch(r"[a-z0-9-]+/[a-z0-9-]+", projected_row["platform"])
            ):
                projected_row["platform"] = "[redacted platform]"
            if "environment" in projected_row:
                environment = projected_row["environment"]
                projected_row["environment"] = sanitize_value(environment) if isinstance(environment, dict) else {}
            if "tool" in projected_row and (
                not isinstance(projected_row["tool"], str)
                or not re.fullmatch(r"[A-Za-z0-9_.-]{1,64}", projected_row["tool"])
            ):
                projected_row["tool"] = "[redacted tool]"
            if "tool_version" in projected_row:
                projected_row["tool_version"] = redact_text(projected_row["tool_version"], limit=256)
            if "tool_version_command" in projected_row:
                version_command = projected_row["tool_version_command"]
                projected_row["tool_version_command"] = (
                    _redact_command_parts(version_command) if isinstance(version_command, list) else []
                )
            if "scope" in projected_row:
                projected_row["scope"] = redact_text(projected_row["scope"], limit=256)
            if "postcondition_commands" in projected_row:
                postconditions = projected_row["postcondition_commands"]
                projected_row["postcondition_commands"] = (
                    [_redact_command_parts(command) for command in postconditions]
                    if isinstance(postconditions, list)
                    else []
                )
            command = projected_row.get("command")
            if not isinstance(command, list) or not command or any(not isinstance(item, str) or not item for item in command):
                if "command" in projected_row:
                    projected_row["command"] = []
            else:
                projected_row["command"] = _redact_command_parts(projected_row["command"])
            if "command" in projected_row and projected_row["command"] and "command_text" not in projected_row:
                projected_row["command_text"] = command_text(projected_row["command"])
            elif "command_text" in projected_row and not isinstance(projected_row["command_text"], str):
                projected_row["command_text"] = None
            elif isinstance(projected_row.get("command_text"), str):
                projected_row["command_text"] = redact_text(projected_row["command_text"], limit=512)
            for key in ("id", "evidence_id", "started_at", "execution_mode", "evidence_type", "acceptance_id"):
                if key in projected_row and not isinstance(projected_row[key], str):
                    projected_row[key] = "[redacted result metadata]"
            if "status" in projected_row and (not isinstance(projected_row["status"], str) or projected_row["status"] not in VALID_RESULT_STATUSES):
                projected_row["status"] = "FAIL"
            if "executed" in projected_row and not isinstance(projected_row["executed"], bool):
                projected_row["executed"] = None
            for key in ("returncode", "outcome", "duration_ms"):
                if key in projected_row and projected_row[key] is not None and not isinstance(projected_row[key], int):
                    projected_row[key] = None
            evidence_path = projected_row.get("evidence_path")
            if not isinstance(evidence_path, str) or any(
                not reference.startswith(("structured-result/", "artifact/"))
                or ".." in reference
                or Path(reference).is_absolute()
                or any(character.isspace() for character in reference)
                for reference in evidence_path.split(",")
            ):
                projected_row["evidence_path"] = "[redacted evidence path]"
            projected_rows.append(projected_row)
        projected["results"] = projected_rows
    elif "results" in projected:
        projected["results"] = []
    if "reason" in projected:
        projected["reason"] = "[redacted result reason]"
    return projected


def command_text(command: list[str]) -> str:
    return shlex.join(_redact_command_parts(command))


def result_row(
    *,
    kind: str,
    name: str,
    status: str,
    command: list[str],
    candidate: dict[str, object],
    started_at: str,
    duration_ms: int = 0,
    returncode: int | None = None,
    stdout: object = "",
    stderr: object = "",
    reason: str | None = None,
    executed: bool | None = None,
    evidence_path: str | None = None,
    execution_mode: str = "process",
    outcome: int | None = None,
    adapter: str = "process",
    platform: str | None = None,
    environment: dict[str, str] | None = None,
    tool: str | None = None,
    tool_version: str | None = None,
    tool_version_command: list[str] | None = None,
    scope: str | None = None,
    postcondition_commands: list[list[str]] | None = None,
) -> dict[str, object]:
    row_id = evidence_id(kind, name, candidate)
    if executed is None:
        executed = status != "BLOCKED / NOT RUN"
    safe_stdout = redact_text(stdout)
    safe_stderr = redact_text(stderr)
    diagnostic = redact_text("\n".join(part for part in (safe_stdout, safe_stderr, redact_text(reason)) if part).strip())
    row: dict[str, object] = {
        "id": row_id,
        "name": name,
        "status": status,
        "executed": executed,
        "execution_mode": execution_mode,
        "candidate": candidate,
        "command": _redact_command_parts(command),
        "command_text": command_text(command),
        "returncode": returncode,
        "outcome": outcome,
        "started_at": started_at,
        "duration_ms": duration_ms,
        "evidence_id": row_id,
        "evidence_path": evidence_path or f"structured-result/{name}",
        "evidence_type": "structured",
        "diagnostic": diagnostic,
        "adapter": adapter,
    }
    if platform:
        row["platform"] = platform
    if environment:
        row["environment"] = sanitize_value(environment)
    if tool:
        row["tool"] = tool
    if tool_version:
        row["tool_version"] = redact_text(tool_version, limit=256)
    if tool_version_command:
        row["tool_version_command"] = _redact_command_parts(tool_version_command)
    if scope:
        row["scope"] = redact_text(scope, limit=256)
    if postcondition_commands:
        row["postcondition_commands"] = [
            _redact_command_parts(command) for command in postcondition_commands
        ]
    return row


def report_for_results(
    *,
    kind: str,
    candidate: dict[str, object],
    results: list[dict[str, object]],
    slice_name: str | None = None,
    evidence_scope: str = "local-quality-checks",
    authentication: dict[str, object] | None = None,
    reason: str | None = None,
) -> dict[str, object]:
    generated_at = utc_now()
    report_id = (
        f"browser-certification:{candidate.get('id')}:{generated_at}"
        if kind == "browser-certification"
        else evidence_id(kind, generated_at, candidate)
    )
    payload: dict[str, object] = {
        "schema_version": SCHEMA_VERSION,
        "kind": kind,
        "slice": slice_name,
        "candidate": candidate,
        "evidence_id": report_id,
        "generated_at": generated_at,
        "evidence_scope": evidence_scope,
        "status": overall_status(results),
        "results": results,
    }
    if authentication is not None:
        payload["authentication"] = authentication
    if reason:
        payload["reason"] = redact_text(reason)
    sanitized = sanitize_result_payload(payload)
    if isinstance(sanitized, dict):
        sanitized_results = sanitized.get("results")
        if isinstance(sanitized_results, list) and len(sanitized_results) == len(results):
            for original, safe in zip(results, sanitized_results):
                if isinstance(original, dict) and isinstance(safe, dict):
                    original.clear()
                    original.update(safe)
            if all(isinstance(item, dict) for item in results):
                sanitized["results"] = results
        return sanitized
    return {
        "schema_version": SCHEMA_VERSION,
        "kind": kind,
        "status": "FAIL",
        "results": [],
        "reason": "result payload could not be safely projected",
    }


def validate_payload(
    payload: object,
    *,
    expected_kind: str,
    expected_candidate: dict[str, object],
    require_authenticated_browser: bool = False,
    source_path: Path | None = None,
    verify_artifacts: bool = True,
    require_command_text: bool = True,
) -> list[str]:
    """Return contract violations; an empty list means the evidence is current."""
    issues: list[str] = []
    if not isinstance(payload, dict):
        return ["result is not an object"]
    if payload.get("schema_version") != SCHEMA_VERSION:
        issues.append("result schema version is invalid")
    if payload.get("kind") != expected_kind:
        issues.append("result kind does not match the requested consumer")
    if expected_kind == FULL_PROFILE_KIND and payload.get("evidence_scope") != FULL_PROFILE_EVIDENCE_SCOPE:
        issues.append("full profile evidence scope is invalid")
    candidate = payload.get("candidate")
    row_candidate = candidate if isinstance(candidate, dict) else expected_candidate
    if not isinstance(candidate, dict) or not candidate.get("commit") or not candidate.get("fingerprint") or not candidate.get("id"):
        issues.append("candidate identity is incomplete")
    elif (
        candidate.get("commit") != expected_candidate.get("commit")
        or candidate.get("fingerprint") != expected_candidate.get("fingerprint")
        or candidate.get("id") != expected_candidate.get("id")
    ):
        issues.append("result candidate is stale or belongs to another working tree")
    if not payload.get("evidence_id") or not payload.get("generated_at"):
        issues.append("top-level evidence identity or timestamp is missing")
    elif isinstance(candidate, dict) and candidate.get("commit"):
        if expected_kind == "browser-certification":
            if str(payload.get("evidence_id")) != f"browser-certification:{candidate.get('id')}:{payload.get('generated_at')}":
                issues.append("browser report identity is not bound to its candidate")
        elif payload.get("evidence_id") != evidence_id(expected_kind, str(payload.get("generated_at")), candidate):
            issues.append("result report identity is not bound to its candidate")
    if require_authenticated_browser:
        authentication = payload.get("authentication")
        if payload.get("status") == "PASS" and (
            payload.get("evidence_scope") != "authenticated-browser"
            or not isinstance(authentication, dict)
            or authentication.get("mode") != "authenticated"
            or not authentication.get("roles")
        ):
            issues.append("browser PASS lacks executed authenticated evidence")
        if payload.get("status") == "PASS":
            preflight = payload.get("preflight")
            if (
                not isinstance(preflight, dict)
                or preflight.get("status") != "PASS"
                or preflight.get("evidence_scope") != "authenticated-browser"
                or preflight.get("executed") is not True
                or not isinstance(preflight.get("evidence_id"), str)
                or not isinstance(preflight.get("candidate"), dict)
                or any(preflight["candidate"].get(key) != candidate.get(key) for key in _CANDIDATE_KEYS)
            ):
                issues.append("browser PASS lacks candidate-bound executed PASS preflight evidence")
        if source_path is not None:
            try:
                artifact_files = [path for path in source_path.parent.rglob("*") if path.is_file()]
                total_bytes = sum(path.stat().st_size for path in artifact_files)
                if total_bytes > MAX_ARTIFACT_TOTAL_BYTES:
                    issues.append("browser evidence directory exceeds the bounded total size")
                if any(path.stat().st_size > MAX_ARTIFACT_FILE_BYTES for path in artifact_files):
                    issues.append("browser evidence directory contains an oversized artifact")
            except OSError:
                issues.append("browser evidence directory could not be inspected")

    rows = payload.get("results")
    if not isinstance(rows, list):
        return issues + ["result rows are not a list"]
    if len(rows) > MAX_RESULT_ROWS:
        issues.append("result contains too many rows")
    row_ids: set[str] = set()
    for row in rows:
        if not isinstance(row, dict):
            issues.append("a result row is not an object")
            continue
        name = row.get("name")
        if not isinstance(name, str) or not name.strip() or name == "[redacted result name]":
            issues.append("a result row lacks a non-empty name")
        status = row.get("status")
        if not isinstance(status, str) or status not in VALID_RESULT_STATUSES:
            issues.append("a result row has an invalid status")
        row_id = row.get("id")
        row_evidence_id = row.get("evidence_id")
        row_identity = row.get("candidate")
        if not isinstance(row_identity, dict) or any(
            row_identity.get(key) != row_candidate.get(key) for key in _CANDIDATE_KEYS
        ):
            issues.append("a result row candidate does not match the report candidate")
        if not isinstance(row_id, str) or not row_id or not isinstance(row_evidence_id, str) or not row_evidence_id:
            issues.append("a result row lacks a stable evidence identity")
        if isinstance(row_id, str) and row_id in row_ids:
            issues.append("result row evidence identities are not unique")
        if isinstance(row_id, str):
            row_ids.add(row_id)
        if expected_kind == "standards-run" and (
            row_id != evidence_id("standards-check", str(row.get("name")), row_candidate)
            or row_evidence_id != evidence_id("standards-check", str(row.get("name")), row_candidate)
        ):
            issues.append("a standards result row is not bound to its candidate identity")
        if expected_kind == "full-validation-profile" and (
            row_id != evidence_id("full-check", str(row.get("name")), row_candidate)
            or row_evidence_id != evidence_id("full-check", str(row.get("name")), row_candidate)
        ):
            issues.append("a full-profile result row is not bound to its candidate identity")
        if expected_kind == "browser-certification" and str(row_evidence_id) != f"{payload.get('evidence_id')}:{row_id}":
            issues.append("a browser result row is not bound to its report identity")
        if require_authenticated_browser and (not isinstance(row_id, str) or row_id != row.get("name") or row_id not in BROWSER_RESULT_NAMES):
            issues.append("browser result row is not an authoritative acceptance row")
        if require_authenticated_browser and row.get("acceptance_id") != row_id:
            issues.append("browser acceptance identity does not match its authoritative row")
        command = row.get("command")
        if not isinstance(command, list) or not command or any(not isinstance(part, str) or not part for part in command):
            issues.append("a result row lacks its executed command/profile")
        if require_command_text and (not isinstance(row.get("command_text"), str) or not row.get("command_text")):
            issues.append("a result row lacks its command text")
        if row.get("command_text") is not None and not isinstance(row.get("command_text"), str):
            issues.append("a result row has invalid command text")
        if (
            isinstance(command, list)
            and command
            and all(isinstance(part, str) and part for part in command)
            and isinstance(row.get("command_text"), str)
            and row.get("command_text") != command_text(command)
        ):
            issues.append("a result row command text does not match its command")
        if not row.get("started_at") or not row.get("evidence_path"):
            issues.append("a result row lacks execution time or evidence path")
        evidence_type = row.get("evidence_type")
        if not isinstance(evidence_type, str) or evidence_type not in {"structured", "artifact"}:
            issues.append("a result row lacks a governed evidence type")
        elif evidence_type == "structured" and str(row.get("evidence_path")) != f"structured-result/{row.get('name')}":
            issues.append("a structured result row has an invalid evidence path")
        elif evidence_type == "artifact":
            references = [reference.strip() for reference in str(row.get("evidence_path")).split(",")]
            for reference in references:
                if not reference.startswith("artifact/") or ".." in reference or Path(reference).is_absolute():
                    issues.append("an artifact result row has an unsafe evidence path")
            if source_path is None:
                if verify_artifacts and status != "BLOCKED / NOT RUN":
                    issues.append("an artifact result row has no source file for evidence verification")
            elif status != "BLOCKED / NOT RUN":
                for reference in references:
                    if not reference.startswith("artifact/") or ".." in reference or Path(reference).is_absolute():
                        continue
                    relative = reference.removeprefix("artifact/")
                    matches = list(source_path.parent.glob(relative))
                    if not matches or not all(match.is_file() for match in matches):
                        issues.append("an artifact result row references missing evidence")
                    elif sum(match.stat().st_size for match in matches) > MAX_ARTIFACT_TOTAL_BYTES or any(match.stat().st_size > MAX_ARTIFACT_FILE_BYTES for match in matches):
                        issues.append("an artifact result row exceeds the bounded evidence size")
        execution_mode = row.get("execution_mode")
        if not isinstance(execution_mode, str) or execution_mode not in {"process", "assertion"}:
            issues.append("a result row lacks an execution mode")
        if not isinstance(row.get("executed"), bool):
            issues.append("a result row lacks explicit executed state")
        elif isinstance(status, str) and status in {"PASS", "FAIL"} and not row.get("executed"):
            issues.append("an executed result row is marked not executed")
        elif status == "BLOCKED / NOT RUN" and row.get("executed"):
            issues.append("a blocked row is marked executed")
        if status == "BLOCKED / NOT RUN" and not str(row.get("diagnostic", "")).strip():
            issues.append("a blocked result row lacks an availability reason")
        if execution_mode == "process" and isinstance(status, str) and status in {"PASS", "FAIL"}:
            if not isinstance(row.get("returncode"), int):
                issues.append("a process result row lacks an exit status")
            elif (status == "PASS" and row.get("returncode") != 0) or (status == "FAIL" and row.get("returncode") == 0):
                issues.append("a process result row contradicts its exit status")
        if execution_mode == "assertion" and isinstance(status, str) and status in {"PASS", "FAIL"}:
            if not isinstance(row.get("outcome"), int):
                issues.append("an assertion result row lacks an outcome status")
            elif (status == "PASS" and row.get("outcome") != 0) or (status == "FAIL" and row.get("outcome") == 0):
                issues.append("an assertion result row contradicts its outcome status")
        if status == "BLOCKED / NOT RUN" and execution_mode == "assertion" and row.get("outcome") is not None:
            issues.append("a blocked assertion row has an outcome status")
        if status == "BLOCKED / NOT RUN" and row.get("returncode") is not None:
            issues.append("a blocked result row has an exit status")
        if status == "BLOCKED / NOT RUN" and row.get("outcome") is not None:
            issues.append("a blocked result row has an outcome status")
        postconditions = row.get("postcondition_commands")
        if postconditions is not None and (
            not isinstance(postconditions, list)
            or any(
                not isinstance(command, list)
                or not command
                or any(not isinstance(part, str) or not part for part in command)
                for command in postconditions
            )
        ):
            issues.append("a result row has malformed postcondition commands")
        duration = row.get("duration_ms")
        if not isinstance(duration, int) or duration < 0:
            issues.append("a result row has an invalid duration")
        if expected_kind == FULL_PROFILE_KIND:
            adapter = row.get("adapter")
            expected_adapter = FULL_PROFILE_CHECK_ADAPTERS.get(str(name))
            if not isinstance(adapter, str) or adapter not in FULL_PROFILE_ADAPTERS:
                issues.append("a full-profile row lacks a valid adapter")
            elif expected_adapter is not None and adapter != expected_adapter:
                issues.append("a full-profile row has the wrong adapter")
            expected_platform = FULL_PROFILE_CHECK_PLATFORMS.get(str(name))
            if expected_platform is not None and row.get("platform") != expected_platform:
                issues.append("a full-profile platform row lacks its declared platform")
            if expected_platform is None and row.get("platform") is not None:
                issues.append("a non-platform full-profile row declares a platform")
            environment = row.get("environment")
            if environment is not None and (
                not isinstance(environment, dict)
                or any(
                    not isinstance(key, str)
                    or not isinstance(value, str)
                    or not re.fullmatch(r"[A-Z][A-Z0-9_]{0,63}", key)
                    for key, value in environment.items()
                )
            ):
                issues.append("a full-profile environment is malformed")
            expected_environment = FULL_PROFILE_CHECK_ENVIRONMENTS.get(str(name))
            if expected_environment is not None and environment != expected_environment:
                issues.append("a cross-platform full-profile row lacks its expected build environment")
            if expected_environment is None and environment is not None:
                issues.append("a non-cross-platform full-profile row declares a build environment")
            if adapter == "docker-runtime" and status == "PASS":
                if not isinstance(postconditions, list) or len(postconditions) != 3:
                    issues.append("a passing container runtime row lacks complete postcondition commands")
                else:
                    state_command, hardening_command, probe_command = postconditions
                    state_name = state_command[-1] if isinstance(state_command, list) and state_command else None
                    if (
                        not isinstance(state_command, list)
                        or state_command[:3] != ["docker", "inspect", "--format"]
                        or len(state_command) != 5
                        or state_command[3] != "{{.State.Status}} {{.State.ExitCode}}"
                        or not isinstance(state_name, str)
                        or not state_name
                    ):
                        issues.append("a passing container runtime row lacks its state postcondition command")
                    if (
                        not isinstance(hardening_command, list)
                        or hardening_command[:3] != ["docker", "inspect", "--format"]
                        or len(hardening_command) != 5
                        or not isinstance(state_name, str)
                        or hardening_command[-1] != state_name
                        or not isinstance(hardening_command[3], str)
                        or ".Config.User" not in hardening_command[3]
                        or ".HostConfig.ReadonlyRootfs" not in hardening_command[3]
                        or ".HostConfig.SecurityOpt" not in hardening_command[3]
                        or ".HostConfig.CapDrop" not in hardening_command[3]
                        or ".HostConfig.Tmpfs" not in hardening_command[3]
                        or ".Mounts" not in hardening_command[3]
                    ):
                        issues.append("a passing container runtime row lacks its hardening postcondition command")
                    if (
                        not isinstance(probe_command, list)
                        or probe_command != ["docker", "exec", state_name, "/app/platform-healthcheck", "--runtime-probe"]
                    ):
                        issues.append("a passing container runtime row lacks its filesystem probe command")
            expected_tool = FULL_PROFILE_SCANNER_TOOLS.get(str(name))
            if expected_tool is not None:
                if row.get("tool") != expected_tool:
                    issues.append("a scanner row lacks its declared tool")
                expected_version_command = FULL_PROFILE_SCANNER_VERSION_COMMANDS[str(name)]
                version_command = row.get("tool_version_command")
                if version_command != expected_version_command:
                    issues.append("a scanner row has the wrong version command")
                if status == "PASS":
                    if not isinstance(row.get("tool_version"), str) or not row.get("tool_version"):
                        issues.append("an executed scanner row lacks tool version evidence")
                expected_scope = FULL_PROFILE_SCANNER_SCOPES[str(name)]
                if row.get("scope") != expected_scope:
                    issues.append("a scanner row lacks its declared scan scope")
        diagnostic = str(row.get("diagnostic", ""))
        if len(diagnostic) > MAX_OUTPUT_CHARS + 32 or len(diagnostic.encode("utf-8", errors="replace")) > MAX_OUTPUT_BYTES:
            issues.append("a result diagnostic exceeds the bounded output limit")

    declared = payload.get("status")
    if not isinstance(declared, str) or declared not in VALID_RESULT_STATUSES:
        issues.append("overall result has an invalid status")
    elif declared != overall_status(rows):
        issues.append("overall status contradicts the row statuses")
    if expected_kind == "standards-run":
        standard_names_list = [row.get("name") for row in rows if isinstance(row, dict) and isinstance(row.get("name"), str)]
        standard_names = set(standard_names_list)
        missing = STANDARDS_RESULT_NAMES - standard_names
        unexpected = standard_names - STANDARDS_RESULT_NAMES
        if missing:
            issues.append("standards result is missing required checks: " + ", ".join(sorted(missing)))
        if unexpected:
            issues.append("standards result contains undeclared checks")
        if len(standard_names_list) != len(standard_names):
            issues.append("standards result contains duplicate checks")
    if expected_kind == "full-validation-profile":
        full_names_list = [row.get("name") for row in rows if isinstance(row, dict) and isinstance(row.get("name"), str)]
        full_names = set(full_names_list)
        missing = FULL_PROFILE_RESULT_NAMES - full_names
        unexpected = full_names - FULL_PROFILE_RESULT_NAMES
        if missing:
            issues.append("full profile is missing required checks: " + ", ".join(sorted(missing)))
        if unexpected:
            issues.append("full profile contains undeclared checks")
        if len(full_names_list) != len(full_names):
            issues.append("full profile contains duplicate checks")
        if len(rows) != len(FULL_PROFILE_RESULT_ORDER):
            issues.append("full profile must contain exactly one row for every required check")
    if require_authenticated_browser and declared == "PASS":
        authentication = payload.get("authentication")
        roles = authentication.get("roles") if isinstance(authentication, dict) else None
        declared_roles = set(roles) if isinstance(roles, list) and all(isinstance(role, str) for role in roles) else set()
        if not isinstance(roles, list) or not all(isinstance(role, str) for role in roles):
            issues.append("browser PASS has malformed authentication roles")
        passing_roles = {
            str(row.get("name", ""))[5:]
            for row in rows
            if str(row.get("name", "")).startswith("ROLE-") and row.get("status") == "PASS" and row.get("executed") is True
        }
        if not {"manager", "contributor", "viewer"} <= declared_roles or not {"manager", "contributor", "viewer"} <= passing_roles:
            issues.append("browser PASS declares roles without corresponding successful role evidence")
        passing_journeys = {
            str(row.get("name", ""))
            for row in rows
            if str(row.get("name", "")) in {f"J{number}" for number in range(1, 14)}
            and row.get("status") == "PASS"
            and row.get("executed") is True
        }
        required_journeys = {f"J{number}" for number in range(1, 14)}
        if not required_journeys <= passing_journeys:
            issues.append("browser PASS lacks successful executed J1-J13 evidence")
        declared_acceptance_names = {
            str(row.get("name"))
            for row in rows
            if isinstance(row.get("name"), str)
        }
        missing_acceptance_names = BROWSER_RESULT_NAMES - declared_acceptance_names
        if missing_acceptance_names:
            issues.append(
                "browser PASS lacks required acceptance rows: "
                + ", ".join(sorted(missing_acceptance_names))
            )
    if expected_kind == "browser-certification-prerequisite-probe":
        if declared != "BLOCKED / NOT RUN" or payload.get("evidence_scope") != "not-run":
            issues.append("prerequisite probe must remain blocked and not-run")
        authentication = payload.get("authentication")
        if not isinstance(authentication, dict) or authentication.get("mode") != "unavailable" or authentication.get("roles") not in ([], None):
            issues.append("prerequisite probe authentication state is invalid")
        if rows:
            issues.append("prerequisite probe must not contain acceptance rows")
    return issues
