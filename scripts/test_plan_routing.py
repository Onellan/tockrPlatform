#!/usr/bin/env python3
"""Contract tests for exact planner-to-router package signatures."""
from __future__ import annotations

import unittest
import re
from pathlib import Path

from validate_plan_routing import validate_plan, validate_text

ROOT = Path(__file__).resolve().parents[1]


class PlanRoutingTests(unittest.TestCase):
    def test_each_package_has_one_compact_route_signature(self):
        plan = """
## WP01 - bounded change
Route: kind=routine; risk=R[UI]

## WP02 - elevated change
Route: kind=authorization; risk=E[AUTH]
"""
        self.assertEqual(validate_text(plan), [])
        self.assertEqual(validate_text(plan, "WP02"), [])

    def test_missing_duplicate_or_unknown_route_is_fail_closed(self):
        plan = """
## WP01 - bounded change
Route: kind=routine; risk=R[]
Route: kind=routine; risk=R[UNKNOWN]
## WP02 - missing route
"""
        issues = validate_text(plan)
        self.assertTrue(any("exactly one" in issue for issue in issues))
        self.assertTrue(any("unsupported risk code" in issue for issue in issues))

    def test_selected_package_ignores_unrelated_terminal_packages(self):
        plan = """
## WP01 - terminal package
No historical route signature is required.

## WP06 - current package
Route: kind=defect; risk=E[GOV,DOC,API]
"""
        self.assertEqual(validate_text(plan, "WP06"), [])
        self.assertTrue(any("exactly one" in issue for issue in validate_text(plan)))

    def test_selected_package_rejects_missing_duplicate_and_unknown(self):
        missing = "## WP06 - current\n"
        duplicate = "## WP06 - current\nRoute: kind=defect; risk=E[GOV]\nRoute: kind=defect; risk=E[DOC]\n"
        valid = "## WP06 - current\nRoute: kind=defect; risk=E[GOV]\n"
        self.assertTrue(any("exactly one" in issue for issue in validate_text(missing, "WP06")))
        self.assertTrue(any("exactly one" in issue for issue in validate_text(duplicate, "WP06")))
        self.assertTrue(any("not declared" in issue for issue in validate_text(valid, "WP99")))

    def test_all_active_execution_plans_are_routeable(self):
        execution_plans = []
        for path in sorted((ROOT / "plan").glob("*.md")):
            text = path.read_text(encoding="utf-8")
            if any(line.startswith(("## WP", "### WP", "### Work package WP")) for line in text.splitlines()):
                execution_plans.append(path)
        # Validate only packages that are still executable. Terminal packages
        # are immutable historical evidence and need no newly-authored route.
        for path in execution_plans:
            text = path.read_text(encoding="utf-8")
            headings = list(re.finditer(r"(?m)^\s{0,3}#{2,6}\s+(?:Work package\s+)?(?P<package>WP[A-Za-z0-9_-]*)\b", text))
            for index, heading in enumerate(headings):
                end = headings[index + 1].start() if index + 1 < len(headings) else len(text)
                section = text[heading.start():end]
                if re.search(r"(?i)Status.{0,24}(?:Terminal|Implemented\s*/\s*terminal)", section):
                    continue
                # A few historical terminal rows predate inline status
                # markers; only executable packages carry a current Route.
                if not re.search(r"(?m)^\s*Route:", section):
                    continue
                self.assertEqual(validate_plan(path, heading.group("package")), [], f"{path} {heading.group('package')}")

    def test_requested_package_must_be_present(self):
        self.assertTrue(any("not declared" in issue for issue in validate_text("## WP01\nRoute: kind=routine; risk=R[]\n", "WP99")))

    def test_duplicate_risk_codes_are_rejected(self):
        issues = validate_text("## WP01\nRoute: kind=routine; risk=R[UI,UI]\n")
        self.assertTrue(any("duplicate risk code" in issue for issue in issues))

    def test_router_plan_input_is_not_optional_in_the_contract(self):
        self.assertTrue(any("exactly one" in issue for issue in validate_text("### Work package WP01 - bounded\n", "WP01")))


if __name__ == "__main__":
    unittest.main()
