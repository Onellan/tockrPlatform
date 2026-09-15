"""Produce a small, non-mutating foundation audit for Tockr Platform."""

from __future__ import annotations

import argparse
import json
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = (
    "architecture.md",
    "docs/architecture/platform-ownership-and-boundaries.md",
    "docs/contracts/platform-contract-v1.md",
    "docs/technical/coding-standards.md",
    "docs/technical/testing-execution-contract.md",
    "plan/active/priority-pf.md",
)


def candidate() -> str:
    result = subprocess.run(["git", "rev-parse", "HEAD"], cwd=ROOT, capture_output=True, text=True, check=False)
    return result.stdout.strip() if result.returncode == 0 else "unresolved"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    missing = [path for path in REQUIRED if not (ROOT / path).exists()]
    plans = sorted((ROOT / "plan/active").glob("pf-b*-s*-*.md"))
    runtime_files = sorted(str(path.relative_to(ROOT)) for path in ROOT.rglob("*.go"))
    report = {
        "schema_version": 1,
        "repository": "tockrPlatform",
        "candidate": candidate(),
        "status": "PASS" if not missing and len(plans) == 21 and not runtime_files else "FAIL",
        "domains": {
            "architecture": "PASS" if not missing else "FAIL",
            "delivery": "PASS" if len(plans) == 21 else "FAIL",
            "runtime": "NOT_APPLICABLE" if not runtime_files else "REVIEW_REQUIRED",
        },
        "missing_foundation_files": missing,
        "slice_plan_count": len(plans),
        "runtime_files": runtime_files,
        "scope_note": "Foundation audit only; no Platform runtime implementation is claimed.",
    }
    payload = json.dumps(report, indent=2) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(payload, encoding="utf-8")
    print(payload, end="")
    return 0 if report["status"] == "PASS" else 1


if __name__ == "__main__":
    raise SystemExit(main())
