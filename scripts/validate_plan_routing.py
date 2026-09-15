#!/usr/bin/env python3
"""Fail-closed validation for planner work-package routing signatures."""
from __future__ import annotations

import argparse
import json
import re
from pathlib import Path

from shared_delivery_contract import ALLOWED_KINDS, ALLOWED_RISK_CODES

ALLOWED_CODES = ALLOWED_RISK_CODES
PACKAGE_HEADING = re.compile(r"^\s{0,3}#{2,6}\s+(?:Work package\s+)?(?P<package>WP[A-Za-z0-9_-]*)\b", re.IGNORECASE)
ROUTE = re.compile(r"^\s*Route:\s*kind=(?P<kind>[a-z-]+);\s*risk=(?P<risk>[REH])\[(?P<codes>[A-Z, ]*)\]\s*$")


def validate_text(text: str, package: str | None = None) -> list[str]:
    issues: list[str] = []
    packages: list[str] = []
    routes: dict[str, list[dict[str, object]]] = {}
    current: str | None = None
    for line_number, line in enumerate(text.splitlines(), 1):
        heading = PACKAGE_HEADING.match(line)
        if heading:
            current = heading.group("package")
            if current in packages:
                issues.append(f"line {line_number}: duplicate work package {current}")
            packages.append(current)
            routes.setdefault(current, [])
            continue
        route = ROUTE.match(line)
        if not route:
            continue
        if current is None:
            issues.append(f"line {line_number}: route signature is outside a work package")
            continue
        raw_codes = [code.strip() for code in route.group("codes").split(",") if code.strip()]
        if len(raw_codes) != len(set(raw_codes)):
            issues.append(f"line {line_number}: duplicate risk code(s) are not allowed")
        codes = sorted(set(raw_codes))
        unknown = sorted(set(codes) - ALLOWED_CODES)
        if route.group("kind") not in ALLOWED_KINDS:
            issues.append(f"line {line_number}: unsupported work-package kind")
        if unknown:
            issues.append(f"line {line_number}: unsupported risk code(s): {', '.join(unknown)}")
        routes[current].append({"kind": route.group("kind"), "risk": route.group("risk"), "risk_codes": codes})

    if not packages:
        issues.append("plan must declare at least one WP work-package heading")
    for name in packages:
        count = len(routes.get(name, []))
        if count != 1:
            issues.append(f"work package {name} must contain exactly one Route signature (found {count})")
    if package is not None:
        if package not in packages:
            issues.append(f"requested work package is not declared: {package}")
        elif len(routes.get(package, [])) != 1:
            issues.append(f"requested work package has no unique Route signature: {package}")
    return issues


def validate_plan(path: Path, package: str | None = None) -> list[str]:
    try:
        text = path.read_text(encoding="utf-8")
    except OSError as error:
        return [f"cannot read plan: {error}"]
    return validate_text(text, package)


def declared_packages(path: Path) -> list[str]:
    """Return declared package IDs in source order."""
    try:
        text = path.read_text(encoding="utf-8")
    except OSError:
        return []
    return [match.group("package") for line in text.splitlines() if (match := PACKAGE_HEADING.match(line))]


def declared_route(path: Path, package: str) -> dict[str, object] | None:
    """Return the unique declared route for a package after file-level checks."""
    try:
        text = path.read_text(encoding="utf-8")
    except OSError:
        return None
    current: str | None = None
    found: list[dict[str, object]] = []
    for line in text.splitlines():
        heading = PACKAGE_HEADING.match(line)
        if heading:
            current = heading.group("package")
            continue
        route = ROUTE.match(line)
        if route and current == package:
            found.append({
                "kind": route.group("kind"),
                "risk": route.group("risk"),
                "risk_codes": sorted({code.strip() for code in route.group("codes").split(",") if code.strip()}),
            })
    return found[0] if len(found) == 1 else None


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("plan", type=Path)
    parser.add_argument("--package")
    args = parser.parse_args()
    issues = validate_plan(args.plan, args.package)
    print(json.dumps({"status": "PASS" if not issues else "FAIL", "issues": issues}, indent=2))
    return 0 if not issues else 1


if __name__ == "__main__":
    raise SystemExit(main())
