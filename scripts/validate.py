"""Repository-native local validation authority for Tockr Platform."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

from validation_registry import FULL_LOCAL, PROFILES

ROOT = Path(__file__).resolve().parents[1]
REQUIRED_FOUNDATION_DOCS = (
    "architecture.md",
    "docs/architecture/platform-ownership-and-boundaries.md",
    "docs/architecture/source-alignment-matrix.md",
    "docs/architecture/versioned-database-migrations.md",
    "docs/contracts/platform-contract-v1.md",
    "docs/technical/coding-standards.md",
    "docs/technical/testing-execution-contract.md",
    "docs/technical/engineering-workflow.md",
    "plan/active/priority-pf.md",
)
SECRET_PATTERNS = (
    re.compile(r"(?i)(password|secret|private[_ -]?key)\s*[:=]\s*['\"][^'\"]+['\"]"),
    re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
)


def candidate() -> str:
    try:
        return subprocess.run(
            ["git", "rev-parse", "HEAD"], cwd=ROOT, text=True, capture_output=True, check=True
        ).stdout.strip()
    except (OSError, subprocess.CalledProcessError):
        return "unresolved"


def foundation_files() -> list[Path]:
    return [ROOT / relative for relative in REQUIRED_FOUNDATION_DOCS]


def run_one(profile: str) -> dict[str, object]:
    if profile not in PROFILES:
        return {"profile": profile, "status": "INVOCATION_FAIL", "reason": "unknown profile"}
    definition = PROFILES[profile]
    if definition["kind"] == "composite":
        rows = [run_one(child) for child in FULL_LOCAL]
        status = "PASS" if all(row["status"] in {"PASS", "NOT_APPLICABLE"} for row in rows) else "FAIL"
        return {"profile": profile, "status": status, "children": rows}
    if definition["kind"] == "runtime":
        prerequisite = ROOT / str(definition["prerequisite"])
        if not prerequisite.exists():
            return {
                "profile": profile,
                "status": "NOT_APPLICABLE",
                "reason": f"runtime prerequisite not introduced yet: {definition['prerequisite']}",
            }
        return {
            "profile": profile,
            "status": "PREREQUISITE_FAIL",
            "reason": "runtime prerequisite exists but no Platform runtime validator is registered",
        }

    if profile == "format":
        missing = [str(path.relative_to(ROOT)) for path in foundation_files() if not path.exists()]
        return {"profile": profile, "status": "PASS" if not missing else "FAIL", "missing": missing}
    if profile == "architecture":
        missing = [str(path.relative_to(ROOT)) for path in foundation_files() if not path.exists()]
        forbidden = [str(path.relative_to(ROOT)) for path in ROOT.rglob("*.go")]
        status = "PASS" if not missing and not forbidden else "FAIL"
        return {"profile": profile, "status": status, "missing": missing, "runtime_files": forbidden}
    if profile == "security":
        hits: list[str] = []
        for path in ROOT.rglob("*.md"):
            text = path.read_text(encoding="utf-8")
            if any(pattern.search(text) for pattern in SECRET_PATTERNS):
                hits.append(str(path.relative_to(ROOT)))
        return {"profile": profile, "status": "PASS" if not hits else "FAIL", "secret_pattern_files": hits}
    if profile == "migration":
        path = ROOT / "docs/architecture/versioned-database-migrations.md"
        text = path.read_text(encoding="utf-8") if path.exists() else ""
        required = ("schema_migrations", "fresh database", "upgrade", "reopen", "fail closed")
        missing = [term for term in required if term.lower() not in text.lower()]
        return {"profile": profile, "status": "PASS" if not missing else "FAIL", "missing": missing}
    if profile == "frontend":
        path = ROOT / "docs/architecture/presentation-architecture-contract.md"
        text = path.read_text(encoding="utf-8") if path.exists() else ""
        required = ("templ", "Tailwind", "server-rendered", "React", "runtime Node.js")
        missing = [term for term in required if term.lower() not in text.lower()]
        return {"profile": profile, "status": "PASS" if not missing else "FAIL", "missing": missing}
    if profile == "quality":
        plans = sorted((ROOT / "plan/active").glob("pf-b*-s*-*.md"))
        routes = sum(path.read_text(encoding="utf-8").count("Route: kind=") for path in plans)
        route_failures = []
        for path in plans:
            checked = subprocess.run(
                [sys.executable, str(ROOT / "scripts/validate_plan_routing.py"), str(path)],
                cwd=ROOT,
                text=True,
                capture_output=True,
                check=False,
            )
            if checked.returncode != 0:
                route_failures.append(str(path.relative_to(ROOT)))
        return {
            "profile": profile,
            "status": "PASS" if len(plans) == 21 and routes >= 63 and not route_failures else "FAIL",
            "slice_plan_count": len(plans),
            "route_signature_count": routes,
            "route_failures": route_failures,
        }
    raise AssertionError(f"unhandled profile: {profile}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)
    subparsers.add_parser("list")
    describe = subparsers.add_parser("describe")
    describe.add_argument("profile")
    run = subparsers.add_parser("run")
    run.add_argument("profile")
    args = parser.parse_args()

    if args.command == "list":
        for profile, definition in PROFILES.items():
            print(f"{profile}: {definition['name']}")
        return 0
    if args.command == "describe":
        if args.profile not in PROFILES:
            print(json.dumps(run_one(args.profile), indent=2))
            return 2
        print(json.dumps({"profile": args.profile, **PROFILES[args.profile]}, indent=2))
        return 0

    result = run_one(args.profile)
    result["candidate"] = candidate()
    print(json.dumps(result, indent=2))
    return 0 if result["status"] in {"PASS", "NOT_APPLICABLE"} else 1


if __name__ == "__main__":
    sys.exit(main())
